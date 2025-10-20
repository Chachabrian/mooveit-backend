package services

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

var (
	s3Session *session.Session
	s3Client  *s3.S3
	uploader  *s3manager.Uploader
	useS3     bool
	baseURL   string
	uploadDir string
)

// InitStorage initializes either S3 or local storage based on configuration
func InitStorage() error {
	// Try to initialize S3
	awsRegion := os.Getenv("AWS_REGION")
	awsAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	awsSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	if awsRegion != "" && awsAccessKey != "" && awsSecretKey != "" {
		// AWS credentials are configured, use S3
		sess, err := session.NewSession(&aws.Config{
			Region: aws.String(awsRegion),
			Credentials: credentials.NewStaticCredentials(
				awsAccessKey,
				awsSecretKey,
				"", // Token (optional)
			),
		})
		if err != nil {
			return fmt.Errorf("failed to create AWS session: %v", err)
		}

		s3Session = sess
		s3Client = s3.New(sess)
		uploader = s3manager.NewUploader(sess)
		useS3 = true

		fmt.Println("✅ AWS S3 storage initialized successfully")
		return nil
	}

	// Fallback to local storage
	useS3 = false
	uploadDir = "/app/uploads"
	baseURL = os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	// Create upload directory
	if err := os.MkdirAll(filepath.Join(uploadDir, "parcels"), 0755); err != nil {
		return fmt.Errorf("failed to create upload directory: %v", err)
	}

	fmt.Println("⚠️  AWS S3 not configured. Using local file storage (not recommended for production)")
	return nil
}

// UploadImage uploads an image to S3 or local storage
func UploadImage(file *multipart.FileHeader, folder string) (string, error) {
	if useS3 {
		return uploadToS3(file, folder)
	}
	return uploadLocally(file, folder)
}

// uploadToS3 uploads a file to AWS S3
func uploadToS3(file *multipart.FileHeader, folder string) (string, error) {
	bucketName := os.Getenv("AWS_S3_BUCKET")
	if bucketName == "" {
		return "", fmt.Errorf("S3 bucket name not configured")
	}

	// Open the file
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer src.Close()

	// Read file content
	buffer := bytes.NewBuffer(nil)
	if _, err := io.Copy(buffer, src); err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	// Detect content type
	contentType := http.DetectContentType(buffer.Bytes())

	// Generate unique filename
	fileExt := filepath.Ext(file.Filename)
	fileName := fmt.Sprintf("%s/%d%s", folder, time.Now().UnixNano(), fileExt)

	// Upload to S3
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(fileName),
		Body:        bytes.NewReader(buffer.Bytes()),
		ContentType: aws.String(contentType),
		// ACL removed - bucket uses bucket policy for public access instead
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %v", err)
	}

	// Construct the public URL manually
	awsRegion := os.Getenv("AWS_REGION")
	publicURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucketName, awsRegion, fileName)

	return publicURL, nil
}

// uploadLocally uploads a file to local storage
func uploadLocally(file *multipart.FileHeader, folder string) (string, error) {
	// Create folder directory
	folderPath := filepath.Join(uploadDir, folder)
	if err := os.MkdirAll(folderPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create folder directory: %v", err)
	}

	// Generate unique filename
	fileExt := filepath.Ext(file.Filename)
	fileName := fmt.Sprintf("%d%s", time.Now().UnixNano(), fileExt)
	filePath := filepath.Join(folderPath, fileName)

	// Open source file
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	defer dst.Close()

	// Copy file
	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to save file: %v", err)
	}

	// Return relative path (will be converted to full URL by handler)
	relativePath := filepath.Join(folder, fileName)
	return relativePath, nil
}

// GetImageURL returns the full URL for an image
// For S3: returns the URL as-is
// For local: prepends the base URL
func GetImageURL(imagePath string) string {
	if useS3 {
		// S3 URLs are already complete
		return imagePath
	}

	// Local files need base URL prepended
	// Replace backslashes with forward slashes for URLs
	imagePath = filepath.ToSlash(imagePath)
	return fmt.Sprintf("%s/uploads/%s", baseURL, imagePath)
}

// DeleteImage deletes an image from S3 or local storage
func DeleteImage(imageURL string) error {
	if useS3 {
		return deleteFromS3(imageURL)
	}
	return deleteLocally(imageURL)
}

// deleteFromS3 deletes a file from AWS S3
func deleteFromS3(fileURL string) error {
	if s3Client == nil {
		return fmt.Errorf("S3 client not initialized")
	}

	bucketName := os.Getenv("AWS_S3_BUCKET")
	if bucketName == "" {
		return fmt.Errorf("S3 bucket name not configured")
	}

	// Extract key from URL
	// Simple extraction - you may need to enhance this
	key := extractKeyFromURL(fileURL)

	_, err := s3Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})

	return err
}

// deleteLocally deletes a file from local storage
func deleteLocally(imageURL string) error {
	// Extract relative path from URL
	// imageURL format: http://localhost:8080/uploads/parcels/123.jpg
	// We need: /app/uploads/parcels/123.jpg

	// This is a simplified implementation
	// You may need to enhance it based on your URL structure
	return nil // Skip actual deletion for now
}

// extractKeyFromURL extracts the S3 key from a full URL
func extractKeyFromURL(url string) string {
	// URL format: https://bucket.s3.region.amazonaws.com/folder/filename
	// Extract: folder/filename

	// Simple implementation - enhance as needed
	parts := filepath.Base(url)
	return parts
}

// IsUsingS3 returns true if S3 storage is being used
func IsUsingS3() bool {
	return useS3
}
