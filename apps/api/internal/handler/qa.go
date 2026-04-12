package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Eahtasham/live-pulse/apps/api/internal/middleware"
	"github.com/Eahtasham/live-pulse/apps/api/internal/models"
	"github.com/Eahtasham/live-pulse/apps/api/internal/service"
)

// QAServiceInterface defines the interface the QA handler depends on.
type QAServiceInterface interface {
	CreateEntry(sessionCode, authorUID, entryType, body string) (*models.QAEntry, error)
	ListEntries(ctx context.Context, sessionCode, cursor string, limit int) ([]models.QAEntry, string, error)
	ModerateEntry(sessionCode string, entryID, hostID uuid.UUID, status string, isHidden *bool) (*models.QAEntry, error)
	GetSessionByCode(code string) (*models.Session, error)
}

type QAHandler struct {
	svc QAServiceInterface
}

func NewQAHandler(svc QAServiceInterface) *QAHandler {
	return &QAHandler{svc: svc}
}

// Request/Response types
type createQARequest struct {
	EntryType string `json:"entry_type"`
	Body      string `json:"body"`
}

type createQAResponse struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id"`
	AuthorUID string `json:"author_uid"`
	EntryType string `json:"entry_type"`
	Body      string `json:"body"`
	Score     int    `json:"score"`
	Status    string `json:"status"`
	IsHidden  bool   `json:"is_hidden"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type moderateQARequest struct {
	Status   string `json:"status,omitempty"`
	IsHidden *bool  `json:"is_hidden,omitempty"`
}

// Create handles POST /v1/sessions/:code/qa
// @Summary Submit a Q&A entry
// @Description Submit a question or comment to a session
// @Tags qa
// @Accept json
// @Produce json
// @Param code path string true "Session code"
// @Param request body createQARequest true "Q&A entry data"
// @Param X-Audience-UID header string true "Audience UID"
// @Success 201 {object} createQAResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/sessions/{code}/qa [post]
func (h *QAHandler) Create(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	// Get audience UID from header
	audienceUID := r.Header.Get("X-Audience-UID")
	if audienceUID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "unauthorized",
			"message": "X-Audience-UID header required",
		})
		return
	}

	var req createQARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "bad_request",
			"message": "invalid request body",
		})
		return
	}

	entry, err := h.svc.CreateEntry(code, audienceUID, req.EntryType, req.Body)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidRequest):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "bad_request",
				"message": err.Error(),
			})
		case errors.Is(err, service.ErrSessionNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error":   "not_found",
				"message": "session not found",
			})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error":   "internal",
				"message": "failed to create entry",
			})
		}
		return
	}

	writeJSON(w, http.StatusCreated, toQAResponse(entry))
}

// List handles GET /v1/sessions/:code/qa
// @Summary List Q&A entries
// @Description List active Q&A entries for a session with cursor pagination
// @Tags qa
// @Accept json
// @Produce json
// @Param code path string true "Session code"
// @Param cursor query string false "Cursor for pagination"
// @Param limit query int false "Limit (default 20, max 100)"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /v1/sessions/{code}/qa [get]
func (h *QAHandler) List(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	// Parse pagination params
	cursor := r.URL.Query().Get("cursor")
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	entries, nextCursor, err := h.svc.ListEntries(r.Context(), code, cursor, limit)
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error":   "not_found",
				"message": "session not found",
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal",
			"message": "failed to list entries",
		})
		return
	}

	result := make([]createQAResponse, len(entries))
	for i, e := range entries {
		result[i] = toQAResponse(&e)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"entries":     result,
		"next_cursor": nextCursor,
	})
}

// Moderate handles PATCH /v1/sessions/:code/qa/:id
// @Summary Moderate a Q&A entry
// @Description Host can pin, answer, hide, or unhide a Q&A entry
// @Tags qa
// @Accept json
// @Produce json
// @Param code path string true "Session code"
// @Param id path string true "Entry ID"
// @Param request body moderateQARequest true "Moderation data"
// @Success 200 {object} createQAResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security Bearer
// @Router /v1/sessions/{code}/qa/{id} [patch]
func (h *QAHandler) Moderate(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	entryIDStr := chi.URLParam(r, "id")

	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "bad_request",
			"message": "invalid entry id",
		})
		return
	}

	// Get host ID from JWT
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "unauthorized",
			"message": "authentication required",
		})
		return
	}
	hostID, err := uuid.Parse(userID)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "unauthorized",
			"message": "invalid user id",
		})
		return
	}

	var req moderateQARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "bad_request",
			"message": "invalid request body",
		})
		return
	}

	entry, err := h.svc.ModerateEntry(code, entryID, hostID, req.Status, req.IsHidden)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidRequest):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "bad_request",
				"message": err.Error(),
			})
		case errors.Is(err, service.ErrSessionNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error":   "not_found",
				"message": "session not found",
			})
		case errors.Is(err, service.ErrQAEntryNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error":   "not_found",
				"message": "entry not found",
			})
		case errors.Is(err, service.ErrNotSessionHost):
			writeJSON(w, http.StatusForbidden, map[string]string{
				"error":   "forbidden",
				"message": "only the host can moderate entries",
			})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error":   "internal",
				"message": "failed to moderate entry",
			})
		}
		return
	}

	writeJSON(w, http.StatusOK, toQAResponse(entry))
}

func toQAResponse(e *models.QAEntry) createQAResponse {
	return createQAResponse{
		ID:        e.ID.String(),
		SessionID: e.SessionID.String(),
		AuthorUID: e.AuthorUID,
		EntryType: e.EntryType,
		Body:      e.Body,
		Score:     e.Score,
		Status:    e.Status,
		IsHidden:  e.IsHidden,
		CreatedAt: e.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: e.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

