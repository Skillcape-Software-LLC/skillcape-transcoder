package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type LocalStorage struct {
	baseDir string
}

func NewLocalStorage(baseDir string) (*LocalStorage, error) {
	// Create directories for uploads and outputs
	dirs := []string{
		filepath.Join(baseDir, "uploads"),
		filepath.Join(baseDir, "outputs"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return &LocalStorage{baseDir: baseDir}, nil
}

// SaveUpload saves an uploaded file and returns the path
func (ls *LocalStorage) SaveUpload(jobID string, filename string, reader io.Reader) (string, error) {
	ext := filepath.Ext(filename)
	savePath := filepath.Join(ls.baseDir, "uploads", jobID+ext)

	file, err := os.Create(savePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		os.Remove(savePath)
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return savePath, nil
}

// GetOutputPath returns the path for a transcoded output file
func (ls *LocalStorage) GetOutputPath(jobID string) string {
	return filepath.Join(ls.baseDir, "outputs", jobID+".mp4")
}

// DeleteFile removes a file from storage
func (ls *LocalStorage) DeleteFile(path string) error {
	if path == "" {
		return nil
	}
	return os.Remove(path)
}

// CleanupJob removes both input and output files for a job
func (ls *LocalStorage) CleanupJob(inputPath, outputPath string) {
	if inputPath != "" {
		os.Remove(inputPath)
	}
	if outputPath != "" {
		os.Remove(outputPath)
	}
}

// FileExists checks if a file exists
func (ls *LocalStorage) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetFileSize returns the size of a file in bytes
func (ls *LocalStorage) GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// OpenFile opens a file for reading
func (ls *LocalStorage) OpenFile(path string) (*os.File, error) {
	return os.Open(path)
}
