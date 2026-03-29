// packages/api-go/internal/utils/r2storage.go
//
// S3-compatible storage utility for Cloudflare R2 and local MinIO.
// Switch between them purely via environment variables — no code changes needed.
//
// Local dev  → R2_ENDPOINT=http://localhost:9000  (MinIO via docker-compose)
// Production → R2_ENDPOINT=https://<ACCOUNT_ID>.r2.cloudflarestorage.com

package utils

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	appconfig "github.com/Mavzz/readpanda/api-go/internal/config"
)

// ObjectStorage holds the S3 client and bucket configuration.
type ObjectStorage struct {
	client     *s3.Client
	bucketName string
	publicURL  string
	endpoint   string
}

var objectStorage *ObjectStorage

// InitObjectStorage initialises the S3-compatible storage client.
func InitObjectStorage(cfg *appconfig.Config) error {
	if cfg.R2Endpoint == "" {
		return fmt.Errorf("R2_ENDPOINT is not set")
	}
	if cfg.R2BucketName == "" {
		return fmt.Errorf("R2_BUCKET_NAME is not set")
	}

	customResolver := aws.EndpointResolverWithOptionsFunc(
		func(service, region string, _ ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               cfg.R2Endpoint,
				SigningRegion:     "auto",
				HostnameImmutable: true,
			}, nil
		},
	)

	awsCfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithRegion("auto"),
		awsconfig.WithEndpointResolverWithOptions(customResolver),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.R2AccessKeyID,
				cfg.R2SecretAccessKey,
				"",
			),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true // required for MinIO and R2
	})

	objectStorage = &ObjectStorage{
		client:     client,
		bucketName: cfg.R2BucketName,
		publicURL:  cfg.R2PublicURL,
		endpoint:   cfg.R2Endpoint,
	}

	fmt.Printf("Object storage (R2/MinIO) initialised — endpoint: %s, bucket: %s\n",
		cfg.R2Endpoint, cfg.R2BucketName)
	return nil
}

// buildPublicURL constructs the public URL for an object key.
func (o *ObjectStorage) buildPublicURL(key string) string {
	base := strings.TrimRight(o.publicURL, "/")
	if base == "" {
		base = strings.TrimRight(o.endpoint, "/") + "/" + o.bucketName
	}
	// If base already ends with the bucket name, don't add it again.
	if !strings.HasSuffix(base, "/"+o.bucketName) {
		base = base + "/" + o.bucketName
	}
	return base + "/" + key
}

// UploadFileToStorage uploads raw bytes to R2 / MinIO and returns the public URL.
func UploadFileToStorage(fileData []byte, contentType, destinationPath string) (string, error) {
	if objectStorage == nil {
		return "", fmt.Errorf("object storage not initialized")
	}
	if len(fileData) == 0 {
		return "", fmt.Errorf("no file data provided for upload")
	}
	if destinationPath == "" {
		return "", fmt.Errorf("no destination path provided for upload")
	}

	_, err := objectStorage.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String(objectStorage.bucketName),
		Key:         aws.String(destinationPath),
		Body:        bytes.NewReader(fileData),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	return objectStorage.buildPublicURL(destinationPath), nil
}

// GetFileDownloadURL generates a presigned URL for a private object.
func GetFileDownloadURL(key string, expiresIn time.Duration) (string, error) {
	if objectStorage == nil {
		return "", fmt.Errorf("object storage not initialized")
	}

	presignClient := s3.NewPresignClient(objectStorage.client)
	req, err := presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(objectStorage.bucketName),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiresIn))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return req.URL, nil
}

// StorageFileInfo holds metadata about an object.
type StorageFileInfo struct {
	Name string
	URL  string
}

// ListFilesFromStorage lists all objects under a given prefix.
func ListFilesFromStorage(prefix string) ([]StorageFileInfo, error) {
	if objectStorage == nil {
		return nil, fmt.Errorf("object storage not initialized")
	}

	resp, err := objectStorage.client.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
		Bucket: aws.String(objectStorage.bucketName),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	var files []StorageFileInfo
	for _, obj := range resp.Contents {
		if strings.HasSuffix(*obj.Key, "/") {
			continue // skip directory placeholders
		}
		files = append(files, StorageFileInfo{
			Name: *obj.Key,
			URL:  objectStorage.buildPublicURL(*obj.Key),
		})
	}
	return files, nil
}
