package jobs

import (
	"context"
	"log"
	"sync"
	"time"
)

type ProcessorFunc func(ctx context.Context, job *Job) error

type WorkerPool struct {
	queue      *Queue
	numWorkers int
	processor  ProcessorFunc
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewWorkerPool(queue *Queue, numWorkers int, processor ProcessorFunc) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		queue:      queue,
		numWorkers: numWorkers,
		processor:  processor,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start launches all workers
func (wp *WorkerPool) Start() {
	log.Printf("Starting worker pool with %d workers", wp.numWorkers)
	for i := 0; i < wp.numWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

// Stop gracefully shuts down all workers
func (wp *WorkerPool) Stop() {
	log.Println("Stopping worker pool...")
	wp.cancel()
	wp.wg.Wait()
	log.Println("Worker pool stopped")
}

func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()
	log.Printf("Worker %d started", id)

	for {
		select {
		case <-wp.ctx.Done():
			log.Printf("Worker %d stopping", id)
			return
		case job, ok := <-wp.queue.Jobs():
			if !ok {
				log.Printf("Worker %d: queue closed", id)
				return
			}
			wp.processJob(id, job)
		}
	}
}

func (wp *WorkerPool) processJob(workerID int, job *Job) {
	log.Printf("Worker %d: processing job %s", workerID, job.ID)
	wp.queue.MarkRunning(job.ID)
	defer wp.queue.MarkDone(job.ID)

	// Create a context with cancellation for this job
	jobCtx, cancel := context.WithCancel(wp.ctx)
	defer cancel()

	start := time.Now()
	err := wp.processor(jobCtx, job)
	duration := time.Since(start)

	if err != nil {
		log.Printf("Worker %d: job %s failed after %v: %v", workerID, job.ID, duration, err)
	} else {
		log.Printf("Worker %d: job %s completed in %v", workerID, job.ID, duration)
	}
}
