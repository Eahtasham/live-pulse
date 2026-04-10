package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Eahtasham/live-pulse/apps/api/internal/middleware"
	"github.com/Eahtasham/live-pulse/apps/api/internal/models"
	"github.com/Eahtasham/live-pulse/apps/api/internal/service"
)

// PollServiceInterface defines the interface the poll handler depends on.
type PollServiceInterface interface {
	CreatePoll(sessionCode string, hostID uuid.UUID, question, answerMode string, timeLimitSec *int, options []models.PollOption) (*models.Poll, error)
	ListPolls(sessionCode string, isHost bool) ([]models.Poll, error)
	GetPoll(sessionCode string, pollID uuid.UUID) (*models.Poll, map[uuid.UUID]int64, error)
	UpdatePollStatus(sessionCode string, pollID, hostID uuid.UUID, newStatus string) (*models.Poll, error)
	GetSessionByCode(code string) (*models.Session, error)
}

type PollHandler struct {
	svc PollServiceInterface
}

func NewPollHandler(svc PollServiceInterface) *PollHandler {
	return &PollHandler{svc: svc}
}

type createPollRequest struct {
	Question     string             `json:"question"`
	AnswerMode   string             `json:"answer_mode"`
	TimeLimitSec *int               `json:"time_limit_sec"`
	Options      []createPollOption `json:"options"`
}

type createPollOption struct {
	Label    string `json:"label"`
	Position int16  `json:"position"`
}

type pollOptionResponse struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Position  int16  `json:"position"`
	VoteCount int64  `json:"vote_count"`
}

type pollResponse struct {
	ID           string               `json:"id"`
	SessionID    string               `json:"session_id"`
	Question     string               `json:"question"`
	AnswerMode   string               `json:"answer_mode"`
	Status       string               `json:"status"`
	TimeLimitSec *int                 `json:"time_limit_sec"`
	Options      []pollOptionResponse `json:"options"`
	CreatedAt    string               `json:"created_at"`
	UpdatedAt    string               `json:"updated_at"`
}

type updatePollRequest struct {
	Status string `json:"status"`
}

func toPollResponse(p *models.Poll, voteCounts map[uuid.UUID]int64) pollResponse {
	opts := make([]pollOptionResponse, len(p.Options))
	for i, o := range p.Options {
		var vc int64
		if voteCounts != nil {
			vc = voteCounts[o.ID]
		}
		opts[i] = pollOptionResponse{
			ID:        o.ID.String(),
			Label:     o.Label,
			Position:  o.Position,
			VoteCount: vc,
		}
	}
	return pollResponse{
		ID:           p.ID.String(),
		SessionID:    p.SessionID.String(),
		Question:     p.Question,
		AnswerMode:   p.AnswerMode,
		Status:       p.Status,
		TimeLimitSec: p.TimeLimitSec,
		Options:      opts,
		CreatedAt:    p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// Create handles POST /v1/sessions/:code/polls
func (h *PollHandler) Create(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	userID := middleware.UserIDFromContext(r.Context())
	hostID, err := uuid.Parse(userID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "bad_request",
			"message": "invalid user id",
		})
		return
	}

	var req createPollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "bad_request",
			"message": "invalid request body",
		})
		return
	}

	// Convert options
	options := make([]models.PollOption, len(req.Options))
	for i, o := range req.Options {
		options[i] = models.PollOption{
			Label:    o.Label,
			Position: o.Position,
		}
	}

	poll, err := h.svc.CreatePoll(code, hostID, req.Question, req.AnswerMode, req.TimeLimitSec, options)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSessionNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error":   "not_found",
				"message": "session not found",
			})
		case errors.Is(err, service.ErrSessionArchived):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "bad_request",
				"message": "cannot create poll in archived session",
			})
		case errors.Is(err, service.ErrNotSessionHost):
			writeJSON(w, http.StatusForbidden, map[string]string{
				"error":   "forbidden",
				"message": "only the session host can create polls",
			})
		case errors.Is(err, service.ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "bad_request",
				"message": err.Error(),
			})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error":   "internal",
				"message": "failed to create poll",
			})
		}
		return
	}

	writeJSON(w, http.StatusCreated, toPollResponse(poll, nil))
}

// List handles GET /v1/sessions/:code/polls
func (h *PollHandler) List(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	// Determine if the requester is the host
	isHost := false
	userID := middleware.UserIDFromContext(r.Context())
	if userID != "" {
		hostID, err := uuid.Parse(userID)
		if err == nil {
			session, err := h.svc.GetSessionByCode(code)
			if err == nil && session.HostID != nil && *session.HostID == hostID {
				isHost = true
			}
		}
	}

	polls, err := h.svc.ListPolls(code, isHost)
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
			"message": "failed to list polls",
		})
		return
	}

	result := make([]pollResponse, len(polls))
	for i, p := range polls {
		result[i] = toPollResponse(&p, nil)
	}

	writeJSON(w, http.StatusOK, result)
}

// Get handles GET /v1/sessions/:code/polls/:pollID
func (h *PollHandler) Get(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	pollIDStr := chi.URLParam(r, "pollID")

	pollID, err := uuid.Parse(pollIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "bad_request",
			"message": "invalid poll id",
		})
		return
	}

	poll, voteCounts, err := h.svc.GetPoll(code, pollID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSessionNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error":   "not_found",
				"message": "session not found",
			})
		case errors.Is(err, service.ErrPollNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error":   "not_found",
				"message": "poll not found",
			})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error":   "internal",
				"message": "failed to get poll",
			})
		}
		return
	}

	writeJSON(w, http.StatusOK, toPollResponse(poll, voteCounts))
}

// Update handles PATCH /v1/sessions/:code/polls/:pollID
func (h *PollHandler) Update(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	pollIDStr := chi.URLParam(r, "pollID")

	userID := middleware.UserIDFromContext(r.Context())
	hostID, err := uuid.Parse(userID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "bad_request",
			"message": "invalid user id",
		})
		return
	}

	pollID, err := uuid.Parse(pollIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "bad_request",
			"message": "invalid poll id",
		})
		return
	}

	var req updatePollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "bad_request",
			"message": "invalid request body",
		})
		return
	}

	poll, err := h.svc.UpdatePollStatus(code, pollID, hostID, req.Status)
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
				"message": "only the session host can modify polls",
			})
		case errors.Is(err, service.ErrPollNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error":   "not_found",
				"message": "poll not found",
			})
		case errors.Is(err, service.ErrInvalidTransition):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "bad_request",
				"message": err.Error(),
			})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error":   "internal",
				"message": "failed to update poll",
			})
		}
		return
	}

	writeJSON(w, http.StatusOK, toPollResponse(poll, nil))
}
