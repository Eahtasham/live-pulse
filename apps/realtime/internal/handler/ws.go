package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"github.com/Eahtasham/live-pulse/apps/realtime/internal/hub"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Restrict origins in production
		return true
	},
}

// WebSocket handles WebSocket upgrade requests.
// @Summary Connect to a session via WebSocket
// @Description Upgrades the HTTP connection to a WebSocket for real-time updates on a session.
// @Description The client will receive broadcast messages for votes, Q&A, and session events.
// @Description
// @Description **Message format (server → client):**
// @Description ```json
// @Description {"event": "vote_update", "data": {...}}
// @Description {"event": "new_question", "data": {...}}
// @Description {"event": "session_closed", "data": {...}}
// @Description ```
// @Description
// @Description **Events:** vote_update, new_question, new_comment, qa_update, session_closed
// @Description
// @Description **Close codes:** 4404 = session not found or not active
// @Tags realtime
// @Param code path string true "Session code (e.g. A1B2C3)"
// @Success 101 {string} string "Switching Protocols"
// @Failure 400 {string} string "Missing session code"
// @Failure 404 {string} string "Session not found or not active"
// @Failure 426 {string} string "WebSocket upgrade required"
// @Router /ws/{code} [get]
func WebSocket(h *hub.Hub, apiBaseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := chi.URLParam(r, "code")
		if code == "" {
			http.Error(w, "missing session code", http.StatusBadRequest)
			return
		}

		// Validate session exists and is active via API service
		if err := validateSession(apiBaseURL, code); err != nil {
			slog.Warn("session validation failed", "code", code, "error", err)
			// If this is already a WebSocket upgrade attempt, reject with close code
			conn, upgradeErr := upgrader.Upgrade(w, r, nil)
			if upgradeErr != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(4404, err.Error()))
			conn.Close()
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("websocket upgrade failed", "code", code, "error", err)
			return
		}

		client := hub.NewClient(h, conn, code)
		h.Register(client)

		go client.WritePump()
		go client.ReadPump()
	}
}

type sessionAPIResponse struct {
	Status string `json:"status"`
}

func validateSession(apiBaseURL, code string) error {
	url := fmt.Sprintf("%s/v1/sessions/%s", apiBaseURL, code)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to validate session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("session not found")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("session validation returned status %d", resp.StatusCode)
	}

	var session sessionAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return fmt.Errorf("failed to decode session response: %w", err)
	}

	if session.Status != "active" {
		return fmt.Errorf("session is %s, not active", session.Status)
	}

	return nil
}
