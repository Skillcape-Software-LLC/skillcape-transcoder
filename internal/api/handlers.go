package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skillcape/transcoder/db"
	"github.com/skillcape/transcoder/internal/jobs"
	"github.com/skillcape/transcoder/internal/storage"
)

type Handler struct {
	localStorage *storage.LocalStorage
	jobQueue     *jobs.Queue
}

func NewHandler(localStorage *storage.LocalStorage, jobQueue *jobs.Queue) *Handler {
	return &Handler{
		localStorage: localStorage,
		jobQueue:     jobQueue,
	}
}

// HealthCheck returns the service health status
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// CreateJob handles video upload and job creation
func (h *Handler) CreateJob(c *gin.Context) {
	// Get the uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "no file uploaded",
		})
		return
	}
	defer file.Close()

	// Generate job ID
	jobID := uuid.New().String()

	// Save the uploaded file
	inputPath, err := h.localStorage.SaveUpload(jobID, header.Filename, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to save uploaded file",
		})
		return
	}

	// Create job record
	job := &jobs.Job{
		ID:           jobID,
		Status:       jobs.StatusPending,
		InputPath:    inputPath,
		OutputPath:   h.localStorage.GetOutputPath(jobID),
		OriginalName: header.Filename,
		Progress:     0,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	// Save to database
	if err := db.CreateJob(job); err != nil {
		h.localStorage.DeleteFile(inputPath)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create job",
		})
		return
	}

	// Enqueue the job
	if err := h.jobQueue.Enqueue(job); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "job queue is full, please try again later",
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"job": job.ToResponse(),
	})
}

// GetJob returns the status of a specific job
func (h *Handler) GetJob(c *gin.Context) {
	jobID := c.Param("id")

	job, err := db.GetJob(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "job not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"job": job.ToResponse(),
	})
}

// ListJobs returns a paginated list of all jobs
func (h *Handler) ListJobs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	jobList, total, err := db.ListJobs(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to list jobs",
		})
		return
	}

	// Convert to response format
	responses := make([]jobs.JobResponse, len(jobList))
	for i, job := range jobList {
		responses[i] = job.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":   responses,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// DeleteJob cancels or deletes a job
func (h *Handler) DeleteJob(c *gin.Context) {
	jobID := c.Param("id")

	job, err := db.GetJob(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "job not found",
		})
		return
	}

	// If job is still running, mark it as cancelled
	if job.Status == jobs.StatusPending || job.Status == jobs.StatusProcessing {
		job.Status = jobs.StatusCancelled
		job.UpdatedAt = time.Now().UTC()
		db.UpdateJob(job)
	}

	// Clean up files
	h.localStorage.CleanupJob(job.InputPath, job.OutputPath)

	// Soft delete from database
	if err := db.DeleteJob(jobID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to delete job",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "job deleted",
	})
}
