package db

import (
	"log"
	"os"
	"path/filepath"

	"github.com/skillcape/transcoder/internal/jobs"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init(dataDir string) error {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	dbPath := filepath.Join(dataDir, "transcoder.db")

	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return err
	}

	// Auto-migrate the schema
	if err := DB.AutoMigrate(&jobs.Job{}); err != nil {
		return err
	}

	log.Printf("Database initialized at %s", dbPath)
	return nil
}

func GetDB() *gorm.DB {
	return DB
}

// CreateJob creates a new job in the database
func CreateJob(job *jobs.Job) error {
	return DB.Create(job).Error
}

// GetJob retrieves a job by ID
func GetJob(id string) (*jobs.Job, error) {
	var job jobs.Job
	if err := DB.First(&job, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// UpdateJob updates an existing job
func UpdateJob(job *jobs.Job) error {
	return DB.Save(job).Error
}

// ListJobs returns all jobs ordered by creation time
func ListJobs(limit, offset int) ([]jobs.Job, int64, error) {
	var jobList []jobs.Job
	var total int64

	DB.Model(&jobs.Job{}).Count(&total)

	err := DB.Order("created_at DESC").Limit(limit).Offset(offset).Find(&jobList).Error
	return jobList, total, err
}

// DeleteJob soft-deletes a job
func DeleteJob(id string) error {
	return DB.Delete(&jobs.Job{}, "id = ?", id).Error
}

// GetPendingJobs returns all jobs with pending status (for recovery after restart)
func GetPendingJobs() ([]jobs.Job, error) {
	var jobList []jobs.Job
	err := DB.Where("status IN ?", []jobs.JobStatus{jobs.StatusPending, jobs.StatusProcessing}).
		Order("created_at ASC").
		Find(&jobList).Error
	return jobList, err
}
