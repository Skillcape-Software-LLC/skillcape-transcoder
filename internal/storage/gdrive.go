package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type GoogleDriveClient struct {
	service  *drive.Service
	folderID string
}

func NewGoogleDriveClient(ctx context.Context, credentialsFile, folderID string) (*GoogleDriveClient, error) {
	// Read credentials file
	credBytes, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	// Create JWT config from service account credentials
	config, err := google.JWTConfigFromJSON(credBytes, drive.DriveFileScope)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	// Create Drive service
	client := config.Client(ctx)
	service, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Drive service: %w", err)
	}

	log.Printf("Google Drive client initialized for folder %s", folderID)
	return &GoogleDriveClient{
		service:  service,
		folderID: folderID,
	}, nil
}

// UploadFile uploads a file to Google Drive and returns the file ID and shareable link
func (gd *GoogleDriveClient) UploadFile(ctx context.Context, filePath, fileName string) (fileID, webViewLink string, err error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create file metadata
	driveFile := &drive.File{
		Name:    fileName,
		Parents: []string{gd.folderID},
	}

	// Upload the file
	uploadedFile, err := gd.service.Files.Create(driveFile).
		Media(file).
		Fields("id, webViewLink").
		Context(ctx).
		Do()
	if err != nil {
		return "", "", fmt.Errorf("failed to upload file: %w", err)
	}

	// Make the file accessible via link
	_, err = gd.service.Permissions.Create(uploadedFile.Id, &drive.Permission{
		Type: "anyone",
		Role: "reader",
	}).Context(ctx).Do()
	if err != nil {
		log.Printf("Warning: failed to set file permissions: %v", err)
	}

	// Get the updated file with webViewLink
	uploadedFile, err = gd.service.Files.Get(uploadedFile.Id).
		Fields("id, webViewLink").
		Context(ctx).
		Do()
	if err != nil {
		return "", "", fmt.Errorf("failed to get file info: %w", err)
	}

	log.Printf("File uploaded to Drive: %s (ID: %s)", fileName, uploadedFile.Id)
	return uploadedFile.Id, uploadedFile.WebViewLink, nil
}

// UploadFileFromReader uploads a file from an io.Reader
func (gd *GoogleDriveClient) UploadFileFromReader(ctx context.Context, reader io.Reader, fileName string) (fileID, webViewLink string, err error) {
	// Create file metadata
	driveFile := &drive.File{
		Name:    fileName,
		Parents: []string{gd.folderID},
	}

	// Upload the file
	uploadedFile, err := gd.service.Files.Create(driveFile).
		Media(reader).
		Fields("id, webViewLink").
		Context(ctx).
		Do()
	if err != nil {
		return "", "", fmt.Errorf("failed to upload file: %w", err)
	}

	// Make the file accessible via link
	_, err = gd.service.Permissions.Create(uploadedFile.Id, &drive.Permission{
		Type: "anyone",
		Role: "reader",
	}).Context(ctx).Do()
	if err != nil {
		log.Printf("Warning: failed to set file permissions: %v", err)
	}

	// Get the updated file with webViewLink
	uploadedFile, err = gd.service.Files.Get(uploadedFile.Id).
		Fields("id, webViewLink").
		Context(ctx).
		Do()
	if err != nil {
		return "", "", fmt.Errorf("failed to get file info: %w", err)
	}

	log.Printf("File uploaded to Drive: %s (ID: %s)", fileName, uploadedFile.Id)
	return uploadedFile.Id, uploadedFile.WebViewLink, nil
}

// DeleteFile removes a file from Google Drive
func (gd *GoogleDriveClient) DeleteFile(ctx context.Context, fileID string) error {
	if fileID == "" {
		return nil
	}
	return gd.service.Files.Delete(fileID).Context(ctx).Do()
}

// GetFileLink returns the shareable link for a file
func (gd *GoogleDriveClient) GetFileLink(ctx context.Context, fileID string) (string, error) {
	file, err := gd.service.Files.Get(fileID).
		Fields("webViewLink").
		Context(ctx).
		Do()
	if err != nil {
		return "", err
	}
	return file.WebViewLink, nil
}
