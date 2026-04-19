package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Eahtasham/live-pulse/apps/api/internal/models"
	"github.com/Eahtasham/live-pulse/apps/api/internal/service"
)

// QAVoteServiceInterface defines the interface the QA vote handler depends on.
type QAVoteServiceInterface interface {
	CastVote(sessionCode string, entryID uuid.UUID, voterUID string, value int16) (*models.QAVote, error)
	GetEntry(sessionCode string, entryID uuid.UUID) (*models.QAEntry, error)
}

type QAVoteHandler struct {
	svc QAVoteServiceInterface
}

func NewQAVoteHandler(svc QAVoteServiceInterface) *QAVoteHandler {
	return &QAVoteHandler{svc: svc}
}

// Request/Response types
type castQAVoteRequest struct {
	AudienceUID string `json:"audience_uid"`
	Value       int16  `json:"value"`
}

type castQAVoteResponse struct {
	ID        string `json:"id,omitempty"`
	EntryID   string `json:"qa_entry_id,omitempty"`
	VoterUID  string `json:"voter_uid,omitempty"`
	VoteValue int16  `json:"vote_value,omitempty"`
	Action    string `json:"action"`
}

// CastVote handles POST /v1/sessions/:code/qa/:id/vote
// @Summary Vote on a Q&A entry
// @Description Cast an upvote or downvote on a question (toggle behavior)
// @Tags qa
// @Accept json
// @Produce json
// @Param code path string true "Session code"
// @Param id path string true "Entry ID"
// @Param request body castQAVoteRequest true "Vote data"
// @Success 200 {object} castQAVoteResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/sessions/{code}/qa/{id}/vote [post]
func (h *QAVoteHandler) CastVote(w http.ResponseWriter, r *http.Request) {
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

	var req castQAVoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "bad_request",
			"message": "invalid request body",
		})
		return
	}

	vote, err := h.svc.CastVote(code, entryID, req.AudienceUID, req.Value)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidVoteValue):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "bad_request",
				"message": "vote value must be 1 (upvote) or -1 (downvote)",
			})
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
		case errors.Is(err, service.ErrQAEntryNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error":   "not_found",
				"message": "entry not found",
			})
		case errors.Is(err, service.ErrQAEntryIsComment):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "bad_request",
				"message": "cannot vote on comments",
			})
		case errors.Is(err, service.ErrQAEntryNotVisible):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "bad_request",
				"message": "entry is not visible",
			})
		case errors.Is(err, service.ErrQAEntryArchived):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "bad_request",
				"message": "entry is archived",
			})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error":   "internal",
				"message": "failed to cast vote",
			})
		}
		return
	}

	// Determine action for response
	action := "voted"
	if vote == nil {
		action = "removed"
	}

	resp := castQAVoteResponse{
		Action: action,
	}
	if vote != nil {
		resp.ID = vote.ID.String()
		resp.EntryID = vote.QAEntryID.String()
		resp.VoterUID = vote.VoterUID
		resp.VoteValue = vote.VoteValue
	}

	writeJSON(w, http.StatusOK, resp)
}
