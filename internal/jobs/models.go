package jobs

import (
	"time"

	"gorm.io/gorm"
)

type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
	StatusCancelled  JobStatus = "cancelled"
)

type Job struct {
	ID            string         `json:"id" gorm:"primaryKey"`
	Status        JobStatus      `json:"status" gorm:"index"`
	InputPath     string         `json:"input_path"`
	OutputPath    string         `json:"output_path,omitempty"`
	DriveURL      string         `json:"drive_url,omitempty"`
	DriveFileID   string         `json:"drive_file_id,omitempty"`
	Progress      int            `json:"progress"`
	Error         string         `json:"error,omitempty"`
	OriginalName  string         `json:"original_name"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	CompletedAt   *time.Time     `json:"completed_at,omitempty"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

type JobResponse struct {
	ID           string     `json:"id"`
	Status       JobStatus  `json:"status"`
	Progress     int        `json:"progress"`
	DriveURL     string     `json:"drive_url,omitempty"`
	Error        string     `json:"error,omitempty"`
	OriginalName string     `json:"original_name"`
	CreatedAt    time.Time  `json:"created_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

func (j *Job) ToResponse() JobResponse {
	return JobResponse{
		ID:           j.ID,
		Status:       j.Status,
		Progress:     j.Progress,
		DriveURL:     j.DriveURL,
		Error:        j.Error,
		OriginalName: j.OriginalName,
		CreatedAt:    j.CreatedAt,
		CompletedAt:  j.CompletedAt,
	}
}

type CreateJobRequest struct {
	WebhookURL string `json:"webhook_url,omitempty"`
}
