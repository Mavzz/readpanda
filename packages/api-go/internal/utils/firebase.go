package utils

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"

	"cloud.google.com/go/storage"
	"github.com/Mavzz/readpanda/api-go/internal/config"
	"google.golang.org/api/option"
)

// FirebaseStorage handles Firebase storage operations
type FirebaseStorage struct {
	bucket     *storage.BucketHandle
	bucketName string
	ctx        context.Context
}

var firebaseStorage *FirebaseStorage

// InitFirebase initializes the Firebase storage client
func InitFirebase(cfg *config.Config) error {
	ctx := context.Background()

	var opt option.ClientOption

	// Prefer service account file if path is set
	if cfg.FirebaseServiceAccountPath != "" {
		if _, err := os.Stat(cfg.FirebaseServiceAccountPath); err != nil {
			return fmt.Errorf("service account file not found: %w", err)
		}
		fmt.Printf("Using service account file: %s\n", cfg.FirebaseServiceAccountPath)
		opt = option.WithCredentialsFile(cfg.FirebaseServiceAccountPath)
	} else {
		// Fall back to constructing credentials from individual env vars
		credJSON := fmt.Sprintf(`{
			"type": "%s",
			"project_id": "%s",
			"private_key_id": "%s",
			"private_key": %s,
			"client_email": "%s",
			"client_id": "%s",
			"auth_uri": "%s",
			"token_uri": "%s",
			"auth_provider_x509_cert_url": "%s",
			"client_x509_cert_url": "%s"
		}`,
			cfg.FirebaseType,
			cfg.FirebaseProjectID,
			cfg.FirebasePrivateKeyID,
			cfg.FirebasePrivateKey,
			cfg.FirebaseClientEmail,
			cfg.FirebaseClientID,
			cfg.FirebaseAuthURI,
			cfg.FirebaseTokenURI,
			cfg.FirebaseAuthProvider,
			cfg.FirebaseCertURL,
		)
		opt = option.WithCredentialsJSON([]byte(credJSON))
	}

	// Create storage client
	client, err := storage.NewClient(ctx, opt)
	if err != nil {
		return fmt.Errorf("error creating Storage client: %w", err)
	}

	// Get bucket
	fmt.Printf("Using Firebase Storage bucket: %s\n", cfg.FirebaseStorageBucket)
	bucket := client.Bucket(cfg.FirebaseStorageBucket)
	if bucket == nil {
		return fmt.Errorf("failed to get storage bucket")
	}

	firebaseStorage = &FirebaseStorage{
		bucket:     bucket,
		bucketName: cfg.FirebaseStorageBucket,
		ctx:        ctx,
	}

	fmt.Println("Firebase Storage initialized successfully.")
	return nil
}

// UploadFileToFirebase uploads a file to Firebase Storage
func UploadFileToFirebase(fileData []byte, contentType, destinationPath string) (string, error) {
	if firebaseStorage == nil {
		return "", fmt.Errorf("Firebase storage not initialized")
	}

	if len(fileData) == 0 {
		return "", fmt.Errorf("no file data provided for upload")
	}

	if destinationPath == "" {
		return "", fmt.Errorf("no destination path provided for upload")
	}

	// Generate a unique token for the file
	token := GenerateUserUID()

	// Get object handle
	obj := firebaseStorage.bucket.Object(destinationPath)

	// Create a writer
	writer := obj.NewWriter(firebaseStorage.ctx)
	writer.ContentType = contentType
	writer.Metadata = map[string]string{
		"firebaseStorageDownloadTokens": token,
	}

	// Write the file data
	if _, err := writer.Write(fileData); err != nil {
		return "", fmt.Errorf("failed to write file data: %w", err)
	}

	// Close the writer
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	// Make the file public
	if err := obj.ACL().Set(firebaseStorage.ctx, storage.AllUsers, storage.RoleReader); err != nil {
		return "", fmt.Errorf("failed to make file public: %w", err)
	}

	// Generate the public URL with token
	bucketName := firebaseStorage.bucketName
	encodedPath := url.QueryEscape(destinationPath)
	publicURL := fmt.Sprintf(
		"https://firebasestorage.googleapis.com/v0/b/%s/o/%s?alt=media&token=%s",
		bucketName, encodedPath, token,
	)

	return publicURL, nil
}

// ReadFileFromFirebase reads a file from Firebase Storage
func ReadFileFromFirebase(filePath string) ([]byte, error) {
	if firebaseStorage == nil {
		return nil, fmt.Errorf("Firebase storage not initialized")
	}

	obj := firebaseStorage.bucket.Object(filePath)
	reader, err := obj.NewReader(firebaseStorage.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create reader: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

// FirebaseFileInfo holds metadata about a file in Firebase Storage
type FirebaseFileInfo struct {
	Name string
	URL  string
}

// ListFilesFromFirebase lists all files under a given prefix in Firebase Storage
func ListFilesFromFirebase(prefix string) ([]FirebaseFileInfo, error) {
	if firebaseStorage == nil {
		return nil, fmt.Errorf("Firebase storage not initialized")
	}

	var files []FirebaseFileInfo
	it := firebaseStorage.bucket.Objects(firebaseStorage.ctx, &storage.Query{Prefix: prefix})
	for {
		attrs, err := it.Next()
		if err != nil {
			break
		}
		// Skip "directory" entries
		if attrs.Name == "" || attrs.Name == prefix {
			continue
		}

		// Build download URL using the download token if present
		token := attrs.Metadata["firebaseStorageDownloadTokens"]
		encodedPath := url.QueryEscape(attrs.Name)
		fileURL := fmt.Sprintf(
			"https://firebasestorage.googleapis.com/v0/b/%s/o/%s?alt=media&token=%s",
			firebaseStorage.bucketName, encodedPath, token,
		)

		files = append(files, FirebaseFileInfo{
			Name: attrs.Name,
			URL:  fileURL,
		})
	}

	return files, nil
}
