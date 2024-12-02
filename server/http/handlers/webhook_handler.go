package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aremxyplug-be/lib/webhook"
)

// EventPayload represents the structure of the incoming webhook payload
type EventPayload struct {
	EventType string `json:"event_type"`
	Data      any    `json:"data"`
}

func WebhookHandler(w http.ResponseWriter, r *http.Request) {

	if !webhook.VerifySignature(r) {
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Parse the incoming JSON payload
	var payload EventPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Process the event
	fmt.Printf("Received event: %s with data: %+v\n", payload.EventType, payload.Data)

	// Respond to the sender
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Event received successfully"))
}
