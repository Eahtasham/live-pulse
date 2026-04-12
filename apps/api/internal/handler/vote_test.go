package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Eahtasham/live-pulse/apps/api/internal/models"
	"github.com/Eahtasham/live-pulse/apps/api/internal/router"
	"github.com/Eahtasham/live-pulse/apps/api/internal/service"
)

// --- mock vote service ---------------------------------------------------

type mockVoteService struct {
	mu        sync.Mutex
	sessions  map[string]*models.Session        // code → session
	polls     map[uuid.UUID]*models.Poll        // pollID → poll
	options   map[uuid.UUID][]models.PollOption // pollID → options
	votes     map[string][]models.Vote          // pollID:audienceUID → votes
	audience  map[string]bool                   // session:uid → exists
}

func newMockVoteService() *mockVoteService {
	return &mockVoteService{
		sessions:  make(map[string]*models.Session),
		polls:     make(map[uuid.UUID]*models.Poll),
		options:   make(map[uuid.UUID][]models.PollOption),
		votes:     make(map[string][]models.Vote),
		audience:  make(map[string]bool),
	}
}

func (m *mockVoteService) addSession(code string, hostID uuid.UUID, status string) *models.Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := &models.Session{
		ID:        uuid.New(),
		HostID:    &hostID,
		Code:      code,
		Title:     "Test Session",
		Status:    status,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.sessions[code] = s
	return s
}

func (m *mockVoteService) addPoll(sessionCode string, question, answerMode, status string, opts []models.PollOption) *models.Poll {
	m.mu.Lock()
	defer m.mu.Unlock()

	session := m.sessions[sessionCode]
	if session == nil {
		return nil
	}

	poll := &models.Poll{
		ID:         uuid.New(),
		SessionID:  session.ID,
		Question:   question,
		AnswerMode: answerMode,
		Status:     status,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Assign IDs to options
	for i := range opts {
		opts[i].ID = uuid.New()
		opts[i].PollID = poll.ID
	}
	poll.Options = opts
	m.options[poll.ID] = opts
	m.polls[poll.ID] = poll
	return poll
}

func (m *mockVoteService) addAudience(sessionCode, uid string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.audience[fmt.Sprintf("%s:%s", sessionCode, uid)] = true
}

func (m *mockVoteService) CastVote(ctx context.Context, sessionCode string, pollID uuid.UUID, optionIDs []uuid.UUID, audienceUID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate audience UID
	if !m.audience[fmt.Sprintf("%s:%s", sessionCode, audienceUID)] {
		return service.ErrInvalidAudienceUID
	}

	// Get session
	session, ok := m.sessions[sessionCode]
	if !ok {
		return service.ErrSessionNotFound
	}

	// Get poll
	poll, ok := m.polls[pollID]
	if !ok || poll.SessionID != session.ID {
		return service.ErrPollNotFound
	}

	// Check poll status
	if poll.Status == "draft" {
		return service.ErrPollNotActive
	}
	if poll.Status == "closed" {
		return service.ErrPollClosed
	}
	if poll.Status != "active" {
		return service.ErrPollNotActive
	}

	// Check answer mode
	if poll.AnswerMode == "single" && len(optionIDs) > 1 {
		return service.ErrSingleModeMultiple
	}

	// Validate options belong to poll
	validOpts := make(map[uuid.UUID]bool)
	for _, opt := range m.options[pollID] {
		validOpts[opt.ID] = true
	}
	for _, optID := range optionIDs {
		if !validOpts[optID] {
			return service.ErrInvalidOption
		}
	}

	// Check for duplicate vote
	key := fmt.Sprintf("%s:%s", pollID, audienceUID)
	if len(m.votes[key]) > 0 {
		return service.ErrDuplicateVote
	}

	// Record votes
	for _, optID := range optionIDs {
		vote := models.Vote{
			ID:          uuid.New(),
			PollID:      pollID,
			OptionID:    optID,
			AudienceUID: audienceUID,
			CreatedAt:   time.Now(),
		}
		m.votes[key] = append(m.votes[key], vote)
	}

	return nil
}

// Helper to get vote counts for a poll
func (m *mockVoteService) getVoteCounts(pollID uuid.UUID) map[uuid.UUID]int64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	counts := make(map[uuid.UUID]int64)
	for key, votes := range m.votes {
		// Check if key starts with pollID
		if len(key) > 36 && key[:36] == pollID.String() {
			for _, v := range votes {
				counts[v.OptionID]++
			}
		}
	}
	return counts
}

// --- tests ----------------------------------------------------------------

func TestCastVote_Success(t *testing.T) {
	voteSvc := newMockVoteService()
	hostID := uuid.New()
	session := voteSvc.addSession("TEST01", hostID, "active")
	voteSvc.addAudience("TEST01", "uid-123")

	poll := voteSvc.addPoll("TEST01", "Favorite color?", "single", "active", []models.PollOption{
		{Label: "Red", Position: 1},
		{Label: "Blue", Position: 2},
	})

	r := router.New(time.Now(), nil, nil, nil, voteSvc, nil, nil, testSecret)

	reqBody := map[string]interface{}{
		"option_ids":   []string{poll.Options[0].ID.String()},
		"audience_uid": "uid-123",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/%s/polls/%s/vote", session.Code, poll.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify vote was recorded
	counts := voteSvc.getVoteCounts(poll.ID)
	if counts[poll.Options[0].ID] != 1 {
		t.Errorf("expected 1 vote for option 0, got %d", counts[poll.Options[0].ID])
	}
}

func TestCastVote_DraftPoll(t *testing.T) {
	voteSvc := newMockVoteService()
	hostID := uuid.New()
	session := voteSvc.addSession("TEST02", hostID, "active")
	voteSvc.addAudience("TEST02", "uid-456")

	poll := voteSvc.addPoll("TEST02", "Draft poll?", "single", "draft", []models.PollOption{
		{Label: "Yes", Position: 1},
	})

	r := router.New(time.Now(), nil, nil, nil, voteSvc, nil, nil, testSecret)

	reqBody := map[string]interface{}{
		"option_ids":   []string{poll.Options[0].ID.String()},
		"audience_uid": "uid-456",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/%s/polls/%s/vote", session.Code, poll.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["message"] != "poll is not active" {
		t.Errorf("expected 'poll is not active', got %s", resp["message"])
	}
}

func TestCastVote_ClosedPoll(t *testing.T) {
	voteSvc := newMockVoteService()
	hostID := uuid.New()
	session := voteSvc.addSession("TEST03", hostID, "active")
	voteSvc.addAudience("TEST03", "uid-789")

	poll := voteSvc.addPoll("TEST03", "Closed poll?", "single", "closed", []models.PollOption{
		{Label: "Yes", Position: 1},
	})

	r := router.New(time.Now(), nil, nil, nil, voteSvc, nil, nil, testSecret)

	reqBody := map[string]interface{}{
		"option_ids":   []string{poll.Options[0].ID.String()},
		"audience_uid": "uid-789",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/%s/polls/%s/vote", session.Code, poll.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["message"] != "poll is closed" {
		t.Errorf("expected 'poll is closed', got %s", resp["message"])
	}
}

func TestCastVote_DuplicateVote(t *testing.T) {
	voteSvc := newMockVoteService()
	hostID := uuid.New()
	session := voteSvc.addSession("TEST04", hostID, "active")
	voteSvc.addAudience("TEST04", "uid-abc")

	poll := voteSvc.addPoll("TEST04", "Vote once?", "single", "active", []models.PollOption{
		{Label: "A", Position: 1},
		{Label: "B", Position: 2},
	})

	r := router.New(time.Now(), nil, nil, nil, voteSvc, nil, nil, testSecret)

	// First vote
	reqBody := map[string]interface{}{
		"option_ids":   []string{poll.Options[0].ID.String()},
		"audience_uid": "uid-abc",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/%s/polls/%s/vote", session.Code, poll.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("first vote should succeed, got %d", rr.Code)
	}

	// Second vote (duplicate)
	reqBody["option_ids"] = []string{poll.Options[1].ID.String()}
	body, _ = json.Marshal(reqBody)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/%s/polls/%s/vote", session.Code, poll.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rr.Code)
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["message"] != "already voted on this poll" {
		t.Errorf("expected 'already voted on this poll', got %s", resp["message"])
	}
}

func TestCastVote_InvalidAudienceUID(t *testing.T) {
	voteSvc := newMockVoteService()
	hostID := uuid.New()
	session := voteSvc.addSession("TEST05", hostID, "active")
	// Don't add audience UID to Redis

	poll := voteSvc.addPoll("TEST05", "Test?", "single", "active", []models.PollOption{
		{Label: "Yes", Position: 1},
	})

	r := router.New(time.Now(), nil, nil, nil, voteSvc, nil, nil, testSecret)

	reqBody := map[string]interface{}{
		"option_ids":   []string{poll.Options[0].ID.String()},
		"audience_uid": "fake-uid",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/%s/polls/%s/vote", session.Code, poll.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestCastVote_SingleModeMultipleOptions(t *testing.T) {
	voteSvc := newMockVoteService()
	hostID := uuid.New()
	session := voteSvc.addSession("TEST06", hostID, "active")
	voteSvc.addAudience("TEST06", "uid-def")

	poll := voteSvc.addPoll("TEST06", "Pick one?", "single", "active", []models.PollOption{
		{Label: "A", Position: 1},
		{Label: "B", Position: 2},
		{Label: "C", Position: 3},
	})

	r := router.New(time.Now(), nil, nil, nil, voteSvc, nil, nil, testSecret)

	// Try to vote for 2 options in single mode
	reqBody := map[string]interface{}{
		"option_ids":   []string{poll.Options[0].ID.String(), poll.Options[1].ID.String()},
		"audience_uid": "uid-def",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/%s/polls/%s/vote", session.Code, poll.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["message"] != "single mode polls only allow one option" {
		t.Errorf("expected 'single mode polls only allow one option', got %s", resp["message"])
	}
}

func TestCastVote_MultiModeMultipleOptions(t *testing.T) {
	voteSvc := newMockVoteService()
	hostID := uuid.New()
	session := voteSvc.addSession("TEST07", hostID, "active")
	voteSvc.addAudience("TEST07", "uid-ghi")

	poll := voteSvc.addPoll("TEST07", "Pick many?", "multi", "active", []models.PollOption{
		{Label: "A", Position: 1},
		{Label: "B", Position: 2},
		{Label: "C", Position: 3},
	})

	r := router.New(time.Now(), nil, nil, nil, voteSvc, nil, nil, testSecret)

	// Vote for 2 options in multi mode
	reqBody := map[string]interface{}{
		"option_ids":   []string{poll.Options[0].ID.String(), poll.Options[1].ID.String()},
		"audience_uid": "uid-ghi",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/%s/polls/%s/vote", session.Code, poll.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify both votes recorded
	counts := voteSvc.getVoteCounts(poll.ID)
	if counts[poll.Options[0].ID] != 1 {
		t.Errorf("expected 1 vote for option 0, got %d", counts[poll.Options[0].ID])
	}
	if counts[poll.Options[1].ID] != 1 {
		t.Errorf("expected 1 vote for option 1, got %d", counts[poll.Options[1].ID])
	}
}

func TestCastVote_InvalidOptionForPoll(t *testing.T) {
	voteSvc := newMockVoteService()
	hostID := uuid.New()
	session := voteSvc.addSession("TEST08", hostID, "active")
	voteSvc.addAudience("TEST08", "uid-jkl")

	poll := voteSvc.addPoll("TEST08", "Test?", "single", "active", []models.PollOption{
		{Label: "Yes", Position: 1},
	})

	// Create another poll with different options
	otherPoll := voteSvc.addPoll("TEST08", "Other?", "single", "active", []models.PollOption{
		{Label: "No", Position: 1},
	})

	r := router.New(time.Now(), nil, nil, nil, voteSvc, nil, nil, testSecret)

	// Try to vote with option from other poll
	reqBody := map[string]interface{}{
		"option_ids":   []string{otherPoll.Options[0].ID.String()},
		"audience_uid": "uid-jkl",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/%s/polls/%s/vote", session.Code, poll.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["message"] != "invalid option for this poll" {
		t.Errorf("expected 'invalid option for this poll', got %s", resp["message"])
	}
}

func TestCastVote_SessionNotFound(t *testing.T) {
	voteSvc := newMockVoteService()

	r := router.New(time.Now(), nil, nil, nil, voteSvc, nil, nil, testSecret)

	reqBody := map[string]interface{}{
		"option_ids":   []string{uuid.New().String()},
		"audience_uid": "uid-xyz",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/INVALID/polls/"+uuid.New().String()+"/vote", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Audience UID is validated first (before DB lookup), so we get 401
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 (invalid audience uid), got %d", rr.Code)
	}
}

func TestCastVote_PollNotFound(t *testing.T) {
	voteSvc := newMockVoteService()
	hostID := uuid.New()
	session := voteSvc.addSession("TEST09", hostID, "active")
	voteSvc.addAudience("TEST09", "uid-mno")

	r := router.New(time.Now(), nil, nil, nil, voteSvc, nil, nil, testSecret)

	reqBody := map[string]interface{}{
		"option_ids":   []string{uuid.New().String()},
		"audience_uid": "uid-mno",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/%s/polls/%s/vote", session.Code, uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestCastVote_TwoDifferentUIDs(t *testing.T) {
	voteSvc := newMockVoteService()
	hostID := uuid.New()
	session := voteSvc.addSession("TEST10", hostID, "active")
	voteSvc.addAudience("TEST10", "uid-1")
	voteSvc.addAudience("TEST10", "uid-2")

	poll := voteSvc.addPoll("TEST10", "Vote?", "single", "active", []models.PollOption{
		{Label: "A", Position: 1},
		{Label: "B", Position: 2},
	})

	r := router.New(time.Now(), nil, nil, nil, voteSvc, nil, nil, testSecret)

	// First user votes for option A
	reqBody1 := map[string]interface{}{
		"option_ids":   []string{poll.Options[0].ID.String()},
		"audience_uid": "uid-1",
	}
	body, _ := json.Marshal(reqBody1)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/%s/polls/%s/vote", session.Code, poll.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("first vote should succeed, got %d", rr.Code)
	}

	// Second user votes for option B
	reqBody2 := map[string]interface{}{
		"option_ids":   []string{poll.Options[1].ID.String()},
		"audience_uid": "uid-2",
	}
	body, _ = json.Marshal(reqBody2)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/%s/polls/%s/vote", session.Code, poll.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("second vote should succeed, got %d", rr.Code)
	}

	// Verify counts
	counts := voteSvc.getVoteCounts(poll.ID)
	if counts[poll.Options[0].ID] != 1 {
		t.Errorf("expected 1 vote for option A, got %d", counts[poll.Options[0].ID])
	}
	if counts[poll.Options[1].ID] != 1 {
		t.Errorf("expected 1 vote for option B, got %d", counts[poll.Options[1].ID])
	}
}

func TestCastVote_VoteCountsAccurate(t *testing.T) {
	voteSvc := newMockVoteService()
	hostID := uuid.New()
	session := voteSvc.addSession("TEST11", hostID, "active")

	poll := voteSvc.addPoll("TEST11", "Vote?", "single", "active", []models.PollOption{
		{Label: "A", Position: 1},
		{Label: "B", Position: 2},
	})

	r := router.New(time.Now(), nil, nil, nil, voteSvc, nil, nil, testSecret)

	// 3 votes for A, 2 votes for B
	for i := 0; i < 3; i++ {
		uid := fmt.Sprintf("voter-a-%d", i)
		voteSvc.addAudience("TEST11", uid)
		reqBody := map[string]interface{}{
			"option_ids":   []string{poll.Options[0].ID.String()},
			"audience_uid": uid,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/%s/polls/%s/vote", session.Code, poll.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("vote %d for A should succeed, got %d", i, rr.Code)
		}
	}

	for i := 0; i < 2; i++ {
		uid := fmt.Sprintf("voter-b-%d", i)
		voteSvc.addAudience("TEST11", uid)
		reqBody := map[string]interface{}{
			"option_ids":   []string{poll.Options[1].ID.String()},
			"audience_uid": uid,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/%s/polls/%s/vote", session.Code, poll.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("vote %d for B should succeed, got %d", i, rr.Code)
		}
	}

	// Verify: A=3, B=2
	counts := voteSvc.getVoteCounts(poll.ID)
	if counts[poll.Options[0].ID] != 3 {
		t.Errorf("expected 3 votes for A, got %d", counts[poll.Options[0].ID])
	}
	if counts[poll.Options[1].ID] != 2 {
		t.Errorf("expected 2 votes for B, got %d", counts[poll.Options[1].ID])
	}
}

func TestCastVote_MissingAudienceUID(t *testing.T) {
	voteSvc := newMockVoteService()
	hostID := uuid.New()
	session := voteSvc.addSession("TEST12", hostID, "active")

	poll := voteSvc.addPoll("TEST12", "Test?", "single", "active", []models.PollOption{
		{Label: "Yes", Position: 1},
	})

	r := router.New(time.Now(), nil, nil, nil, voteSvc, nil, nil, testSecret)

	reqBody := map[string]interface{}{
		"option_ids": []string{poll.Options[0].ID.String()},
		// Missing audience_uid
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/%s/polls/%s/vote", session.Code, poll.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestCastVote_MissingOptionIDs(t *testing.T) {
	voteSvc := newMockVoteService()
	hostID := uuid.New()
	session := voteSvc.addSession("TEST13", hostID, "active")
	voteSvc.addAudience("TEST13", "uid-zzz")

	poll := voteSvc.addPoll("TEST13", "Test?", "single", "active", []models.PollOption{
		{Label: "Yes", Position: 1},
	})

	r := router.New(time.Now(), nil, nil, nil, voteSvc, nil, nil, testSecret)

	reqBody := map[string]interface{}{
		"audience_uid": "uid-zzz",
		// Missing option_ids
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/%s/polls/%s/vote", session.Code, poll.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestCastVote_InvalidPollID(t *testing.T) {
	voteSvc := newMockVoteService()
	hostID := uuid.New()
	session := voteSvc.addSession("TEST14", hostID, "active")

	r := router.New(time.Now(), nil, nil, nil, voteSvc, nil, nil, testSecret)

	reqBody := map[string]interface{}{
		"option_ids":   []string{uuid.New().String()},
		"audience_uid": "uid-123",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/%s/polls/invalid-uuid/vote", session.Code), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestCastVote_InvalidOptionID(t *testing.T) {
	voteSvc := newMockVoteService()
	hostID := uuid.New()
	session := voteSvc.addSession("TEST15", hostID, "active")
	voteSvc.addAudience("TEST15", "uid-www")

	poll := voteSvc.addPoll("TEST15", "Test?", "single", "active", []models.PollOption{
		{Label: "Yes", Position: 1},
	})

	r := router.New(time.Now(), nil, nil, nil, voteSvc, nil, nil, testSecret)

	reqBody := map[string]interface{}{
		"option_ids":   []string{"invalid-uuid"},
		"audience_uid": "uid-www",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/%s/polls/%s/vote", session.Code, poll.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

