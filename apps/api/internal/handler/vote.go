package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Eahtasham/live-pulse/apps/api/internal/service"
)

// VoteServiceInterface defines the interface the vote handler depends on.
type VoteServiceInterface interface {
	CastVote(ctx context.Context, sessionCode string, pollID uuid.UUID, optionIDs []uuid.UUID, audienceUID string) error
}

type VoteHandler struct {
	svc VoteServiceInterface
}

func NewVoteHandler(svc VoteServiceInterface) *VoteHandler {
	return &VoteHandler{svc: svc}
}

type castVoteRequest struct {
	OptionIDs   []string `json:"option_ids"`
	AudienceUID string   `json:"audience_uid"`
}

type castVoteResponse struct {
	Message string `json:"message"`
}

// CastVote handles POST /v1/sessions/:code/polls/:pollID/vote
// @Summary Cast a vote on a poll
// @Description Submit a vote on an active poll (audience only, requires valid audience UID)
// @Tags votes
// @Accept json
// @Produce json
// @Param code path string true "Session code"
// @Param pollID path string true "Poll ID"
// @Param request body castVoteRequest true "Vote data"
// @Success 200 {object} castVoteResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /v1/sessions/{code}/polls/{pollID}/vote [post]
func (h *VoteHandler) CastVote(w http.ResponseWriter, r *http.Request) {
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

	var req castVoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "bad_request",
			"message": "invalid request body",
		})
		return
	}

	// Validate required fields
	if req.AudienceUID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "bad_request",
			"message": "audience_uid is required",
		})
		return
	}

	if len(req.OptionIDs) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "bad_request",
			"message": "option_ids is required",
		})
		return
	}

	// Parse option IDs
	optionIDs := make([]uuid.UUID, 0, len(req.OptionIDs))
	for _, idStr := range req.OptionIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "bad_request",
				"message": "invalid option_id: " + idStr,
			})
			return
		}
		optionIDs = append(optionIDs, id)
	}

	// Call service
	if err := h.svc.CastVote(r.Context(), code, pollID, optionIDs, req.AudienceUID); err != nil {
		switch {
		case errors.Is(err, service.ErrSessionNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error":   "not_found",
				"message": "session not found",
			})
		case errors.Is(err, service.ErrSessionArchived):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "bad_request",
				"message": "Session is archived",
			})
		case errors.Is(err, service.ErrPollNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error":   "not_found",
				"message": "poll not found",
			})
		case errors.Is(err, service.ErrPollNotActive):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "bad_request",
				"message": "poll is not active",
			})
		case errors.Is(err, service.ErrPollClosed):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "bad_request",
				"message": "poll is closed",
			})
		case errors.Is(err, service.ErrInvalidOption):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "bad_request",
				"message": "invalid option for this poll",
			})
		case errors.Is(err, service.ErrDuplicateVote):
			writeJSON(w, http.StatusConflict, map[string]string{
				"error":   "conflict",
				"message": "already voted on this poll",
			})
		case errors.Is(err, service.ErrInvalidAudienceUID):
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error":   "unauthorized",
				"message": "invalid audience uid",
			})
		case errors.Is(err, service.ErrSingleModeMultiple):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "bad_request",
				"message": "single mode polls only allow one option",
			})
		case errors.Is(err, service.ErrNoOptions):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "bad_request",
				"message": "no options provided",
			})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error":   "internal",
				"message": "failed to cast vote",
			})
		}
		return
	}

	writeJSON(w, http.StatusOK, castVoteResponse{
		Message: "vote recorded",
	})
}
