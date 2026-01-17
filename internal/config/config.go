package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port                  string
	APIKey                string
	WorkerCount           int
	TempDir               string
	GoogleCredentialsFile string
	GoogleDriveFolderID   string
	WebhookURL            string
	WebhookRetryCount     int
}

func Load() *Config {
	return &Config{
		Port:                  getEnv("PORT", "8080"),
		APIKey:                getEnv("API_KEY", ""),
		WorkerCount:           getEnvInt("WORKER_COUNT", 2),
		TempDir:               getEnv("TEMP_DIR", "/tmp/transcoder"),
		GoogleCredentialsFile: getEnv("GOOGLE_CREDENTIALS_FILE", "/config/credentials.json"),
		GoogleDriveFolderID:   getEnv("GOOGLE_DRIVE_FOLDER_ID", ""),
		WebhookURL:            getEnv("WEBHOOK_URL", ""),
		WebhookRetryCount:     getEnvInt("WEBHOOK_RETRY_COUNT", 3),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
