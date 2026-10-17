package utils

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	fcm "google.golang.org/api/fcm/v1"
	"google.golang.org/api/option"
)

// The request SendPush makes, as a stand-in FCM sees it.
func TestSendPushPlaysTheDefaultSound(t *testing.T) {
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"name": "projects/p/messages/1"}`)
	}))
	defer srv.Close()

	service, err := fcm.NewService(context.Background(),
		option.WithEndpoint(srv.URL), option.WithoutAuthentication(), option.WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}
	original := push
	push = &pushClient{service: service, projectID: "p"}
	t.Cleanup(func() { push = original })

	err = SendPush(context.Background(), "tok", PushMessage{
		Title: "AI Learning", Body: "bob joined the room", Data: map[string]string{"type": "ROOM_JOIN"},
	})
	if err != nil {
		t.Fatal(err)
	}

	var sent struct {
		Message struct {
			Notification struct{ Title, Body string }
			Data         map[string]string
			Apns         struct {
				Payload struct {
					Aps struct {
						Sound string `json:"sound"`
					} `json:"aps"`
				} `json:"payload"`
			} `json:"apns"`
		} `json:"message"`
	}
	if err := json.Unmarshal(body, &sent); err != nil {
		t.Fatalf("decoding %s: %v", body, err)
	}
	if sent.Message.Apns.Payload.Aps.Sound != "default" {
		t.Errorf("no default sound in %s", body)
	}
	if sent.Message.Notification.Title != "AI Learning" || sent.Message.Data["type"] != "ROOM_JOIN" {
		t.Errorf("title or data lost: %s", body)
	}
}
