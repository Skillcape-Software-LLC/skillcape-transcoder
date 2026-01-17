package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Client struct {
	httpClient *http.Client
	retryCount int
}

type Payload struct {
	JobID        string `json:"job_id"`
	Status       string `json:"status"`
	DriveURL     string `json:"drive_url,omitempty"`
	DriveFileID  string `json:"drive_file_id,omitempty"`
	Error        string `json:"error,omitempty"`
	OriginalName string `json:"original_name"`
	CompletedAt  string `json:"completed_at"`
}

func NewClient(retryCount int) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		retryCount: retryCount,
	}
}

// Send sends a webhook notification with retry logic
func (c *Client) Send(ctx context.Context, url string, payload *Payload) error {
	if url == "" {
		log.Printf("No webhook URL configured, skipping notification for job %s", payload.JobID)
		return nil
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= c.retryCount; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s, 8s...
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			log.Printf("Webhook retry %d/%d for job %s in %v", attempt, c.retryCount, payload.JobID, backoff)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		err := c.sendRequest(ctx, url, jsonData)
		if err == nil {
			log.Printf("Webhook sent successfully for job %s", payload.JobID)
			return nil
		}

		lastErr = err
		log.Printf("Webhook attempt %d failed for job %s: %v", attempt+1, payload.JobID, err)
	}

	return fmt.Errorf("webhook failed after %d attempts: %w", c.retryCount+1, lastErr)
}

func (c *Client) sendRequest(ctx context.Context, url string, jsonData []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Skillcape-Transcoder/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("webhook returned status %d", resp.StatusCode)
}

// SendAsync sends a webhook notification asynchronously
func (c *Client) SendAsync(url string, payload *Payload) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if err := c.Send(ctx, url, payload); err != nil {
			log.Printf("Async webhook failed for job %s: %v", payload.JobID, err)
		}
	}()
}
