package jobs

import (
	"log"
	"sync"
)

type Queue struct {
	jobs    chan *Job
	mu      sync.RWMutex
	running map[string]bool
}

func NewQueue(bufferSize int) *Queue {
	return &Queue{
		jobs:    make(chan *Job, bufferSize),
		running: make(map[string]bool),
	}
}

// Enqueue adds a job to the queue
func (q *Queue) Enqueue(job *Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	select {
	case q.jobs <- job:
		log.Printf("Job %s enqueued", job.ID)
		return nil
	default:
		return ErrQueueFull
	}
}

// Dequeue retrieves the next job from the queue (blocking)
func (q *Queue) Dequeue() *Job {
	return <-q.jobs
}

// Jobs returns the job channel for workers to consume
func (q *Queue) Jobs() <-chan *Job {
	return q.jobs
}

// MarkRunning marks a job as currently being processed
func (q *Queue) MarkRunning(jobID string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.running[jobID] = true
}

// MarkDone removes a job from the running set
func (q *Queue) MarkDone(jobID string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.running, jobID)
}

// IsRunning checks if a job is currently being processed
func (q *Queue) IsRunning(jobID string) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.running[jobID]
}

// Size returns the current number of jobs in the queue
func (q *Queue) Size() int {
	return len(q.jobs)
}

// Close closes the job queue channel
func (q *Queue) Close() {
	close(q.jobs)
}

// Custom errors
type QueueError string

func (e QueueError) Error() string {
	return string(e)
}

const (
	ErrQueueFull    QueueError = "job queue is full"
	ErrJobNotFound  QueueError = "job not found"
	ErrJobCancelled QueueError = "job was cancelled"
)
