package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/skillcape/transcoder/db"
	"github.com/skillcape/transcoder/internal/api"
	"github.com/skillcape/transcoder/internal/config"
	"github.com/skillcape/transcoder/internal/jobs"
	"github.com/skillcape/transcoder/internal/storage"
	"github.com/skillcape/transcoder/internal/transcoder"
	"github.com/skillcape/transcoder/internal/webhook"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting Skillcape Transcoder...")

	// Load configuration
	cfg := config.Load()

	// Check FFmpeg availability
	if !transcoder.IsFFmpegAvailable() {
		log.Fatal("FFmpeg is not installed or not in PATH")
	}
	log.Println("FFmpeg detected")

	// Initialize database
	if err := db.Init(cfg.TempDir); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize local storage
	localStorage, err := storage.NewLocalStorage(cfg.TempDir)
	if err != nil {
		log.Fatalf("Failed to initialize local storage: %v", err)
	}

	// Initialize Google Drive client (optional - continues if credentials not found)
	var driveClient *storage.GoogleDriveClient
	if cfg.GoogleCredentialsFile != "" && cfg.GoogleDriveFolderID != "" {
		driveClient, err = storage.NewGoogleDriveClient(
			context.Background(),
			cfg.GoogleCredentialsFile,
			cfg.GoogleDriveFolderID,
		)
		if err != nil {
			log.Printf("Warning: Google Drive not configured: %v", err)
		}
	} else {
		log.Println("Google Drive integration not configured")
	}

	// Initialize webhook client
	webhookClient := webhook.NewClient(cfg.WebhookRetryCount)

	// Create job queue
	jobQueue := jobs.NewQueue(100) // Buffer size of 100 jobs

	// Create job processor
	processor := createJobProcessor(cfg, localStorage, driveClient, webhookClient)

	// Create and start worker pool
	workerPool := jobs.NewWorkerPool(jobQueue, cfg.WorkerCount, processor)
	workerPool.Start()

	// Recover pending jobs from database
	recoverPendingJobs(jobQueue)

	// Setup HTTP router
	router := api.SetupRouter(cfg, localStorage, jobQueue)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server listening on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Stop worker pool
	workerPool.Stop()

	log.Println("Server exited")
}

func createJobProcessor(
	cfg *config.Config,
	localStorage *storage.LocalStorage,
	driveClient *storage.GoogleDriveClient,
	webhookClient *webhook.Client,
) jobs.ProcessorFunc {
	return func(ctx context.Context, job *jobs.Job) error {
		// Update job status to processing
		job.Status = jobs.StatusProcessing
		job.UpdatedAt = time.Now().UTC()
		db.UpdateJob(job)

		// Create progress callback
		progressCallback := func(progress int) {
			job.Progress = progress
			job.UpdatedAt = time.Now().UTC()
			db.UpdateJob(job)
		}

		// Transcode the video
		ffmpeg := transcoder.New(job.InputPath, job.OutputPath)
		ffmpeg.OnProgress(progressCallback)

		if err := ffmpeg.Transcode(ctx); err != nil {
			return handleJobFailure(job, webhookClient, cfg.WebhookURL, fmt.Sprintf("transcoding failed: %v", err))
		}

		// Upload to Google Drive if configured
		if driveClient != nil {
			outputName := job.OriginalName
			if len(outputName) > 4 {
				outputName = outputName[:len(outputName)-4] + ".mp4"
			} else {
				outputName = job.ID + ".mp4"
			}

			fileID, webViewLink, err := driveClient.UploadFile(ctx, job.OutputPath, outputName)
			if err != nil {
				return handleJobFailure(job, webhookClient, cfg.WebhookURL, fmt.Sprintf("drive upload failed: %v", err))
			}

			job.DriveFileID = fileID
			job.DriveURL = webViewLink
		}

		// Mark as completed
		now := time.Now().UTC()
		job.Status = jobs.StatusCompleted
		job.Progress = 100
		job.CompletedAt = &now
		job.UpdatedAt = now
		db.UpdateJob(job)

		// Clean up local files after successful upload
		if driveClient != nil {
			localStorage.CleanupJob(job.InputPath, job.OutputPath)
		}

		// Send webhook notification
		webhookClient.SendAsync(cfg.WebhookURL, &webhook.Payload{
			JobID:        job.ID,
			Status:       string(job.Status),
			DriveURL:     job.DriveURL,
			DriveFileID:  job.DriveFileID,
			OriginalName: job.OriginalName,
			CompletedAt:  now.Format(time.RFC3339),
		})

		return nil
	}
}

func handleJobFailure(job *jobs.Job, webhookClient *webhook.Client, webhookURL, errMsg string) error {
	log.Printf("Job %s failed: %s", job.ID, errMsg)

	now := time.Now().UTC()
	job.Status = jobs.StatusFailed
	job.Error = errMsg
	job.CompletedAt = &now
	job.UpdatedAt = now
	db.UpdateJob(job)

	// Send failure webhook
	webhookClient.SendAsync(webhookURL, &webhook.Payload{
		JobID:        job.ID,
		Status:       string(job.Status),
		Error:        errMsg,
		OriginalName: job.OriginalName,
		CompletedAt:  now.Format(time.RFC3339),
	})

	return fmt.Errorf(errMsg)
}

func recoverPendingJobs(jobQueue *jobs.Queue) {
	pendingJobs, err := db.GetPendingJobs()
	if err != nil {
		log.Printf("Warning: failed to recover pending jobs: %v", err)
		return
	}

	if len(pendingJobs) == 0 {
		return
	}

	log.Printf("Recovering %d pending jobs", len(pendingJobs))
	for i := range pendingJobs {
		job := &pendingJobs[i]
		// Reset status to pending for re-processing
		job.Status = jobs.StatusPending
		job.Progress = 0
		db.UpdateJob(job)

		if err := jobQueue.Enqueue(job); err != nil {
			log.Printf("Failed to re-enqueue job %s: %v", job.ID, err)
		}
	}
}
