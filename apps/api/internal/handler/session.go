package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Eahtasham/live-pulse/apps/api/internal/middleware"
	"github.com/Eahtasham/live-pulse/apps/api/internal/models"
	"github.com/Eahtasham/live-pulse/apps/api/internal/service"
)

// SessionServiceInterface defines the interface the session handler depends on.
type SessionServiceInterface interface {
	CreateSession(hostID uuid.UUID, title string) (*models.Session, error)
	ListSessionsByHost(hostID uuid.UUID) ([]models.Session, error)
	GetSessionByCode(code string) (*models.Session, error)
	JoinSession(ctx context.Context, code, clientID string) (string, *models.Session, error)
	CloseSession(ctx context.Context, code string, hostID uuid.UUID) (*models.Session, error)
}

type SessionHandler struct {
	svc SessionServiceInterface
}

func NewSessionHandler(svc SessionServiceInterface) *SessionHandler {
	return &SessionHandler{svc: svc}
}

type createSessionRequest struct {
	Title string `json:"title"`
}

type createSessionResponse struct {
	ID        string `json:"id"`
	Code      string `json:"code"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

type joinSessionResponse struct {
	AudienceUID  string `json:"audience_uid"`
	SessionTitle string `json:"session_title"`
}

// Create handles POST /v1/sessions
// @Summary Create a new session
// @Description Create a new polling session for the authenticated host
// @Tags sessions
// @Accept json
// @Produce json
// @Param request body createSessionRequest true "Session creation request"
// @Success 201 {object} createSessionResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Security Bearer
// @Router /v1/sessions [post]
func (h *SessionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "bad_request",
			"message": "invalid request body",
		})
		return
	}

	if req.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "bad_request",
			"message": "title is required",
		})
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	hostID, err := uuid.Parse(userID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "bad_request",
			"message": "invalid user id",
		})
		return
	}

	session, err := h.svc.CreateSession(hostID, req.Title)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal",
			"message": "failed to create session",
		})
		return
	}

	writeJSON(w, http.StatusCreated, createSessionResponse{
		ID:        session.ID.String(),
		Code:      session.Code,
		Title:     session.Title,
		Status:    session.Status,
		CreatedAt: session.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// List handles GET /v1/sessions (authenticated)
// @Summary List sessions for host
// @Description Get all sessions created by the authenticated host
// @Tags sessions
// @Accept json
// @Produce json
// @Success 200 {array} models.Session
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security Bearer
// @Router /v1/sessions [get]
func (h *SessionHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	hostID, err := uuid.Parse(userID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "bad_request",
			"message": "invalid user id",
		})
		return
	}

	sessions, err := h.svc.ListSessionsByHost(hostID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal",
			"message": "failed to list sessions",
		})
		return
	}

	writeJSON(w, http.StatusOK, sessions)
}

// GetByCode handles GET /v1/sessions/:code (public)
// @Summary Get session by code
// @Description Get a session by its unique code (public access)
// @Tags sessions
// @Accept json
// @Produce json
// @Param code path string true "Session code"
// @Success 200 {object} models.Session
// @Failure 404 {object} map[string]string
// @Router /v1/sessions/{code} [get]
func (h *SessionHandler) GetByCode(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "bad_request",
			"message": "code is required",
		})
		return
	}

	session, err := h.svc.GetSessionByCode(code)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":   "not_found",
			"message": "session not found",
		})
		return
	}

	writeJSON(w, http.StatusOK, session)
}

// Join handles POST /v1/sessions/:code/join (public)
// @Summary Join a session
// @Description Join a session as an audience member and get an audience UID
// @Tags sessions
// @Accept json
// @Produce json
// @Param code path string true "Session code"
// @Param X-Client-ID header string false "Client ID for session tracking"
// @Success 200 {object} joinSessionResponse
// @Failure 404 {object} map[string]string
// @Router /v1/sessions/{code}/join [post]
func (h *SessionHandler) Join(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "bad_request",
			"message": "code is required",
		})
		return
	}

	clientID := r.Header.Get("X-Client-ID")

	uid, session, err := h.svc.JoinSession(r.Context(), code, clientID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":   "not_found",
			"message": "session not found",
		})
		return
	}

	writeJSON(w, http.StatusOK, joinSessionResponse{
		AudienceUID:  uid,
		SessionTitle: session.Title,
	})
}

func (h *SessionHandler) Close(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "bad_request",
			"message": "code is required",
		})
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	hostID, err := uuid.Parse(userID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "bad_request",
			"message": "invalid user id",
		})
		return
	}

	session, err := h.svc.CloseSession(r.Context(), code, hostID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSessionNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error":   "not_found",
				"message": "session not found",
			})
		case errors.Is(err, service.ErrNotSessionHost):
			writeJSON(w, http.StatusForbidden, map[string]string{
				"error":   "forbidden",
				"message": "only the session host can close the session",
			})
		case errors.Is(err, service.ErrSessionArchived):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "bad_request",
				"message": "session is already archived",
			})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error":   "internal",
				"message": "failed to close session",
			})
		}
		return
	}

	writeJSON(w, http.StatusOK, session)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
