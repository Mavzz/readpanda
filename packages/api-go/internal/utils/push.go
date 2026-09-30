package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/Mavzz/readpanda/api-go/internal/config"
	fcm "google.golang.org/api/fcm/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

// ErrPushDisabled is returned by SendPush when InitPush didn't succeed —
// callers treat it as "nothing to do", not as a failure.
var ErrPushDisabled = errors.New("push notifications are not configured")

// ErrStaleToken means FCM no longer recognises the device token (the app was
// uninstalled, or the token rotated). The caller should forget it.
var ErrStaleToken = errors.New("device token is no longer registered")

type pushClient struct {
	service   *fcm.Service
	projectID string
}

var push *pushClient

// InitPush initialises the Firebase Cloud Messaging client from the same
// service account the rest of the Firebase config uses.
func InitPush(cfg *config.Config) error {
	projectID, err := firebaseProjectID(cfg)
	if err != nil {
		return err
	}

	credentials, err := FirebaseCredentialOption(cfg)
	if err != nil {
		return err
	}

	service, err := fcm.NewService(
		context.Background(),
		credentials,
		option.WithScopes(fcm.FirebaseMessagingScope),
	)
	if err != nil {
		return fmt.Errorf("error creating FCM client: %w", err)
	}

	push = &pushClient{service: service, projectID: projectID}
	fmt.Printf("Push notifications enabled for Firebase project %s.\n", projectID)
	return nil
}

// firebaseProjectID reads FIREBASE_PROJECT_ID, falling back to the project_id
// inside the service account file.
func firebaseProjectID(cfg *config.Config) (string, error) {
	if cfg.FirebaseProjectID != "" {
		return cfg.FirebaseProjectID, nil
	}
	if cfg.FirebaseServiceAccountPath == "" {
		return "", errors.New("FIREBASE_PROJECT_ID or FIREBASE_SERVICE_ACCOUNT_PATH is required")
	}
	raw, err := os.ReadFile(cfg.FirebaseServiceAccountPath)
	if err != nil {
		return "", fmt.Errorf("reading service account file: %w", err)
	}
	var account struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal(raw, &account); err != nil || account.ProjectID == "" {
		return "", errors.New("service account file has no project_id")
	}
	return account.ProjectID, nil
}

// PushMessage is one notification as the device shows it, plus the data the
// app reads when it is tapped.
type PushMessage struct {
	Title string
	Body  string
	Data  map[string]string
}

// SendPush delivers msg to a single device token. It returns ErrStaleToken
// when FCM says the token is dead, so the caller can delete it.
func SendPush(ctx context.Context, token string, msg PushMessage) error {
	if push == nil {
		return ErrPushDisabled
	}

	request := &fcm.SendMessageRequest{
		Message: &fcm.Message{
			Token: token,
			Notification: &fcm.Notification{
				Title: msg.Title,
				Body:  msg.Body,
			},
			Data: msg.Data,
		},
	}

	_, err := push.service.Projects.Messages.
		Send("projects/"+push.projectID, request).
		Context(ctx).
		Do()
	if err == nil {
		return nil
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		// UNREGISTERED comes back as 404; a token that was never valid as a
		// 400 naming the registration token. Anything else is our problem
		// (credentials, payload, quota), not the token's.
		if apiErr.Code == http.StatusNotFound ||
			(apiErr.Code == http.StatusBadRequest && strings.Contains(strings.ToLower(apiErr.Message), "registration token")) {
			return ErrStaleToken
		}
	}
	return err
}
