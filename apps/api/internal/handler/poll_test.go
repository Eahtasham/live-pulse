package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Eahtasham/live-pulse/apps/api/internal/models"
	"github.com/Eahtasham/live-pulse/apps/api/internal/router"
	"github.com/Eahtasham/live-pulse/apps/api/internal/service"
)

// --- mock poll service ---------------------------------------------------

type mockPollService struct {
	mu        sync.Mutex
	sessions  map[string]*models.Session // code → session
	polls     map[uuid.UUID]*models.Poll // pollID → poll
	bySession map[uuid.UUID][]uuid.UUID  // sessionID → []pollID
}

func newMockPollService() *mockPollService {
	return &mockPollService{
		sessions:  make(map[string]*models.Session),
		polls:     make(map[uuid.UUID]*models.Poll),
		bySession: make(map[uuid.UUID][]uuid.UUID),
	}
}

func (m *mockPollService) addSession(code string, hostID uuid.UUID, status string) *models.Session {
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

func (m *mockPollService) GetSessionByCode(code string) (*models.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[code]
	if !ok {
		return nil, service.ErrSessionNotFound
	}
	return s, nil
}

func (m *mockPollService) CreatePoll(sessionCode string, hostID uuid.UUID, question, answerMode string, timeLimitSec *int, options []models.PollOption) (*models.Poll, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.sessions[sessionCode]
	if !ok {
		return nil, service.ErrSessionNotFound
	}
	if s.Status == "archived" {
		return nil, service.ErrSessionArchived
	}
	if s.HostID == nil || *s.HostID != hostID {
		return nil, service.ErrNotSessionHost
	}

	// Validate
	if question == "" || len(question) > 500 {
		return nil, fmt.Errorf("%w: question is required and must be at most 500 characters", service.ErrInvalidInput)
	}
	if answerMode != "single" && answerMode != "multi" {
		return nil, fmt.Errorf("%w: answer_mode must be 'single' or 'multi'", service.ErrInvalidInput)
	}
	if len(options) < 2 || len(options) > 6 {
		return nil, fmt.Errorf("%w: options must be between 2 and 6", service.ErrInvalidInput)
	}
	for _, o := range options {
		if o.Label == "" || len(o.Label) > 200 {
			return nil, fmt.Errorf("%w: option label is required and must be at most 200 characters", service.ErrInvalidInput)
		}
	}
	if timeLimitSec != nil && *timeLimitSec <= 0 {
		return nil, fmt.Errorf("%w: time_limit_sec must be a positive integer", service.ErrInvalidInput)
	}

	now := time.Now()
	poll := &models.Poll{
		ID:           uuid.New(),
		SessionID:    s.ID,
		Question:     question,
		AnswerMode:   answerMode,
		TimeLimitSec: timeLimitSec,
		Status:       "draft",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Assign IDs to options and sort by position
	for i := range options {
		options[i].ID = uuid.New()
		options[i].PollID = poll.ID
	}
	sort.Slice(options, func(i, j int) bool {
		return options[i].Position < options[j].Position
	})
	poll.Options = options

	m.polls[poll.ID] = poll
	m.bySession[s.ID] = append(m.bySession[s.ID], poll.ID)

	return poll, nil
}

func (m *mockPollService) ListPolls(sessionCode string, isHost bool) ([]models.Poll, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.sessions[sessionCode]
	if !ok {
		return nil, service.ErrSessionNotFound
	}

	ids := m.bySession[s.ID]
	var result []models.Poll
	for i := len(ids) - 1; i >= 0; i-- {
		p := m.polls[ids[i]]
		if !isHost && p.Status == "draft" {
			continue
		}
		result = append(result, *p)
	}
	return result, nil
}

func (m *mockPollService) GetPoll(sessionCode string, pollID uuid.UUID) (*models.Poll, map[uuid.UUID]int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.sessions[sessionCode]
	if !ok {
		return nil, nil, service.ErrSessionNotFound
	}

	p, ok := m.polls[pollID]
	if !ok {
		return nil, nil, service.ErrPollNotFound
	}

	// Return zero vote counts
	voteCounts := make(map[uuid.UUID]int64)
	for _, o := range p.Options {
		voteCounts[o.ID] = 0
	}
	return p, voteCounts, nil
}

func (m *mockPollService) UpdatePollStatus(sessionCode string, pollID, hostID uuid.UUID, newStatus string) (*models.Poll, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.sessions[sessionCode]
	if !ok {
		return nil, service.ErrSessionNotFound
	}
	if s.HostID == nil || *s.HostID != hostID {
		return nil, service.ErrNotSessionHost
	}

	p, ok := m.polls[pollID]
	if !ok {
		return nil, service.ErrPollNotFound
	}

	// Validate transition
	transitions := map[string]string{"draft": "active", "active": "closed"}
	allowed, validFrom := transitions[p.Status]
	if !validFrom || allowed != newStatus {
		return nil, fmt.Errorf("%w: cannot transition from %s to %s", service.ErrInvalidTransition, p.Status, newStatus)
	}

	p.Status = newStatus
	p.UpdatedAt = time.Now()
	return p, nil
}

// --- helper to set up router with poll mock ---

func setupRouterWithPolls(t *testing.T) (*mockPollService, http.Handler) {
	t.Helper()
	pollSvc := newMockPollService()
	return pollSvc, router.New(time.Now(), nil, nil, pollSvc, nil, nil, nil, testSecret)
}

// --- acceptance tests ---------------------------------------------------

// Create poll with valid data → 201, returns poll with options
func TestCreatePoll_Success(t *testing.T) {
	pollSvc, r := setupRouterWithPolls(t)
	hostID := uuid.New()
	pollSvc.addSession("A1B2C3", hostID, "active")

	token := generateTestJWT(t, hostID.String(), "host@example.com", testSecret, 24*time.Hour)
	body := `{
		"question": "What is the time complexity of binary search?",
		"answer_mode": "single",
		"time_limit_sec": 30,
		"options": [
			{"label": "O(1)", "position": 0},
			{"label": "O(log n)", "position": 1},
			{"label": "O(n)", "position": 2},
			{"label": "O(n log n)", "position": 3}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/A1B2C3/polls", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp["id"] == nil || resp["id"] == "" {
		t.Error("expected non-empty id")
	}
	if resp["question"] != "What is the time complexity of binary search?" {
		t.Errorf("expected question match, got %v", resp["question"])
	}
	if resp["answer_mode"] != "single" {
		t.Errorf("expected answer_mode=single, got %v", resp["answer_mode"])
	}
	if resp["status"] != "draft" {
		t.Errorf("expected status=draft, got %v", resp["status"])
	}

	opts, ok := resp["options"].([]interface{})
	if !ok || len(opts) != 4 {
		t.Fatalf("expected 4 options, got %v", resp["options"])
	}
}

// Poll is created with status: "draft" by default
func TestCreatePoll_DefaultDraftStatus(t *testing.T) {
	pollSvc, r := setupRouterWithPolls(t)
	hostID := uuid.New()
	pollSvc.addSession("AABBCC", hostID, "active")

	token := generateTestJWT(t, hostID.String(), "host@example.com", testSecret, 24*time.Hour)
	body := `{
		"question": "Test?",
		"answer_mode": "single",
		"options": [
			{"label": "A", "position": 0},
			{"label": "B", "position": 1}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/AABBCC/polls", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp["status"] != "draft" {
		t.Errorf("expected status=draft, got %v", resp["status"])
	}
}

// Cannot create a poll with fewer than 2 options → 400
func TestCreatePoll_TooFewOptions_Returns400(t *testing.T) {
	pollSvc, r := setupRouterWithPolls(t)
	hostID := uuid.New()
	pollSvc.addSession("FEWOPT", hostID, "active")

	token := generateTestJWT(t, hostID.String(), "host@example.com", testSecret, 24*time.Hour)
	body := `{
		"question": "Test?",
		"answer_mode": "single",
		"options": [{"label": "Only one", "position": 0}]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/FEWOPT/polls", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

// Cannot create a poll with more than 6 options → 400
func TestCreatePoll_TooManyOptions_Returns400(t *testing.T) {
	pollSvc, r := setupRouterWithPolls(t)
	hostID := uuid.New()
	pollSvc.addSession("MNYOPT", hostID, "active")

	token := generateTestJWT(t, hostID.String(), "host@example.com", testSecret, 24*time.Hour)
	body := `{
		"question": "Test?",
		"answer_mode": "single",
		"options": [
			{"label": "A", "position": 0},
			{"label": "B", "position": 1},
			{"label": "C", "position": 2},
			{"label": "D", "position": 3},
			{"label": "E", "position": 4},
			{"label": "F", "position": 5},
			{"label": "G", "position": 6}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/MNYOPT/polls", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

// GET /v1/sessions/:code/polls as audience → returns only active and closed polls (NOT draft)
func TestListPolls_AudienceSeesOnlyActiveAndClosed(t *testing.T) {
	pollSvc, r := setupRouterWithPolls(t)
	hostID := uuid.New()
	pollSvc.addSession("LSTPOL", hostID, "active")

	token := generateTestJWT(t, hostID.String(), "host@example.com", testSecret, 24*time.Hour)

	// Create 3 polls
	for _, q := range []string{"Draft Poll", "Active Poll", "Closed Poll"} {
		body := fmt.Sprintf(`{
			"question": "%s",
			"answer_mode": "single",
			"options": [
				{"label": "A", "position": 0},
				{"label": "B", "position": 1}
			]
		}`, q)
		req := httptest.NewRequest(http.MethodPost, "/v1/sessions/LSTPOL/polls", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create poll %q: expected 201, got %d", q, rec.Code)
		}
	}

	// Activate second poll
	pollSvc.mu.Lock()
	var pollIDs []uuid.UUID
	for _, s := range pollSvc.sessions {
		pollIDs = pollSvc.bySession[s.ID]
	}
	pollSvc.mu.Unlock()

	// Activate poll[1]
	activateBody := `{"status": "active"}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/sessions/LSTPOL/polls/"+pollIDs[1].String(), bytes.NewBufferString(activateBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("activate poll: expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	// Close poll[2]: first activate, then close
	req = httptest.NewRequest(http.MethodPatch, "/v1/sessions/LSTPOL/polls/"+pollIDs[2].String(), bytes.NewBufferString(activateBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("activate poll for close: expected 200, got %d", rec.Code)
	}

	closeBody := `{"status": "closed"}`
	req = httptest.NewRequest(http.MethodPatch, "/v1/sessions/LSTPOL/polls/"+pollIDs[2].String(), bytes.NewBufferString(closeBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("close poll: expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	// Now GET as audience (no auth) — should only see 2 polls (active + closed), NOT draft
	req = httptest.NewRequest(http.MethodGet, "/v1/sessions/LSTPOL/polls", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("list polls: expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var polls []map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&polls)

	if len(polls) != 2 {
		t.Fatalf("expected 2 polls (active+closed), got %d", len(polls))
	}

	for _, p := range polls {
		status := p["status"].(string)
		if status == "draft" {
			t.Errorf("audience should not see draft polls, got status=%s", status)
		}
	}
}

// PATCH to activate → status changes to active
func TestUpdatePoll_Activate(t *testing.T) {
	pollSvc, r := setupRouterWithPolls(t)
	hostID := uuid.New()
	pollSvc.addSession("ACTPOL", hostID, "active")

	token := generateTestJWT(t, hostID.String(), "host@example.com", testSecret, 24*time.Hour)

	// Create poll
	body := `{"question":"Q?","answer_mode":"single","options":[{"label":"A","position":0},{"label":"B","position":1}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/ACTPOL/polls", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	pollID := created["id"].(string)

	// Activate
	req = httptest.NewRequest(http.MethodPatch, "/v1/sessions/ACTPOL/polls/"+pollID, bytes.NewBufferString(`{"status":"active"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["status"] != "active" {
		t.Errorf("expected status=active, got %v", resp["status"])
	}
}

// PATCH to close → status changes to closed
func TestUpdatePoll_Close(t *testing.T) {
	pollSvc, r := setupRouterWithPolls(t)
	hostID := uuid.New()
	pollSvc.addSession("CLSPOL", hostID, "active")

	token := generateTestJWT(t, hostID.String(), "host@example.com", testSecret, 24*time.Hour)

	// Create and activate
	body := `{"question":"Q?","answer_mode":"single","options":[{"label":"A","position":0},{"label":"B","position":1}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/CLSPOL/polls", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	pollID := created["id"].(string)

	// Activate first
	req = httptest.NewRequest(http.MethodPatch, "/v1/sessions/CLSPOL/polls/"+pollID, bytes.NewBufferString(`{"status":"active"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	// Close
	req = httptest.NewRequest(http.MethodPatch, "/v1/sessions/CLSPOL/polls/"+pollID, bytes.NewBufferString(`{"status":"closed"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["status"] != "closed" {
		t.Errorf("expected status=closed, got %v", resp["status"])
	}
}

// Cannot transition from closed → active → 400
func TestUpdatePoll_ClosedToActive_Returns400(t *testing.T) {
	pollSvc, r := setupRouterWithPolls(t)
	hostID := uuid.New()
	pollSvc.addSession("CLSACT", hostID, "active")

	token := generateTestJWT(t, hostID.String(), "host@example.com", testSecret, 24*time.Hour)

	// Create → activate → close
	body := `{"question":"Q?","answer_mode":"single","options":[{"label":"A","position":0},{"label":"B","position":1}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/CLSACT/polls", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	pollID := created["id"].(string)

	for _, status := range []string{"active", "closed"} {
		req = httptest.NewRequest(http.MethodPatch, "/v1/sessions/CLSACT/polls/"+pollID, bytes.NewBufferString(fmt.Sprintf(`{"status":"%s"}`, status)))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec = httptest.NewRecorder()
		r.ServeHTTP(rec, req)
	}

	// Try closed → active
	req = httptest.NewRequest(http.MethodPatch, "/v1/sessions/CLSACT/polls/"+pollID, bytes.NewBufferString(`{"status":"active"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

// Cannot transition from closed → draft → 400
func TestUpdatePoll_ClosedToDraft_Returns400(t *testing.T) {
	pollSvc, r := setupRouterWithPolls(t)
	hostID := uuid.New()
	pollSvc.addSession("CLSDRF", hostID, "active")

	token := generateTestJWT(t, hostID.String(), "host@example.com", testSecret, 24*time.Hour)

	body := `{"question":"Q?","answer_mode":"single","options":[{"label":"A","position":0},{"label":"B","position":1}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/CLSDRF/polls", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	pollID := created["id"].(string)

	// activate then close
	for _, status := range []string{"active", "closed"} {
		req = httptest.NewRequest(http.MethodPatch, "/v1/sessions/CLSDRF/polls/"+pollID, bytes.NewBufferString(fmt.Sprintf(`{"status":"%s"}`, status)))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec = httptest.NewRecorder()
		r.ServeHTTP(rec, req)
	}

	// Try closed → draft
	req = httptest.NewRequest(http.MethodPatch, "/v1/sessions/CLSDRF/polls/"+pollID, bytes.NewBufferString(`{"status":"draft"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

// Options are returned in position order
func TestCreatePoll_OptionsInPositionOrder(t *testing.T) {
	pollSvc, r := setupRouterWithPolls(t)
	hostID := uuid.New()
	pollSvc.addSession("ORDOPT", hostID, "active")

	token := generateTestJWT(t, hostID.String(), "host@example.com", testSecret, 24*time.Hour)
	// Send options in reverse order
	body := `{
		"question": "Order test?",
		"answer_mode": "single",
		"options": [
			{"label": "Third", "position": 2},
			{"label": "First", "position": 0},
			{"label": "Second", "position": 1}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/ORDOPT/polls", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)

	opts := resp["options"].([]interface{})
	labels := make([]string, len(opts))
	for i, o := range opts {
		labels[i] = o.(map[string]interface{})["label"].(string)
	}

	expected := []string{"First", "Second", "Third"}
	for i, label := range labels {
		if label != expected[i] {
			t.Errorf("option %d: expected %q, got %q", i, expected[i], label)
		}
	}
}

// Only the session host can create polls → other users get 403
func TestCreatePoll_NonHost_Returns403(t *testing.T) {
	pollSvc, r := setupRouterWithPolls(t)
	hostID := uuid.New()
	pollSvc.addSession("NOHOST", hostID, "active")

	// Different user
	otherID := uuid.New()
	token := generateTestJWT(t, otherID.String(), "other@example.com", testSecret, 24*time.Hour)

	body := `{"question":"Q?","answer_mode":"single","options":[{"label":"A","position":0},{"label":"B","position":1}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/NOHOST/polls", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

// Only the session host can activate/close polls → other users get 403
func TestUpdatePoll_NonHost_Returns403(t *testing.T) {
	pollSvc, r := setupRouterWithPolls(t)
	hostID := uuid.New()
	pollSvc.addSession("NOHST2", hostID, "active")

	hostToken := generateTestJWT(t, hostID.String(), "host@example.com", testSecret, 24*time.Hour)

	// Create as host
	body := `{"question":"Q?","answer_mode":"single","options":[{"label":"A","position":0},{"label":"B","position":1}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/NOHST2/polls", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+hostToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	pollID := created["id"].(string)

	// Try activate as different user
	otherID := uuid.New()
	otherToken := generateTestJWT(t, otherID.String(), "other@example.com", testSecret, 24*time.Hour)

	req = httptest.NewRequest(http.MethodPatch, "/v1/sessions/NOHST2/polls/"+pollID, bytes.NewBufferString(`{"status":"active"}`))
	req.Header.Set("Authorization", "Bearer "+otherToken)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

// Creating a poll for a non-existent session → 404
func TestCreatePoll_NonExistentSession_Returns404(t *testing.T) {
	_, r := setupRouterWithPolls(t)

	token := generateTestJWT(t, uuid.New().String(), "host@example.com", testSecret, 24*time.Hour)
	body := `{"question":"Q?","answer_mode":"single","options":[{"label":"A","position":0},{"label":"B","position":1}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/NOEXST/polls", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

// Creating a poll for an archived session → 400
func TestCreatePoll_ArchivedSession_Returns400(t *testing.T) {
	pollSvc, r := setupRouterWithPolls(t)
	hostID := uuid.New()
	pollSvc.addSession("ARCHVD", hostID, "archived")

	token := generateTestJWT(t, hostID.String(), "host@example.com", testSecret, 24*time.Hour)
	body := `{"question":"Q?","answer_mode":"single","options":[{"label":"A","position":0},{"label":"B","position":1}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/ARCHVD/polls", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

// GET poll details with vote counts per option
func TestGetPoll_WithVoteCounts(t *testing.T) {
	pollSvc, r := setupRouterWithPolls(t)
	hostID := uuid.New()
	pollSvc.addSession("GETPOL", hostID, "active")

	token := generateTestJWT(t, hostID.String(), "host@example.com", testSecret, 24*time.Hour)

	body := `{"question":"Q?","answer_mode":"single","options":[{"label":"A","position":0},{"label":"B","position":1}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/GETPOL/polls", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	pollID := created["id"].(string)

	// GET poll details (public, no auth)
	req = httptest.NewRequest(http.MethodGet, "/v1/sessions/GETPOL/polls/"+pollID, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)

	opts := resp["options"].([]interface{})
	for _, o := range opts {
		opt := o.(map[string]interface{})
		if _, ok := opt["vote_count"]; !ok {
			t.Error("expected vote_count in option response")
		}
	}
}

// Suppress unused import warnings
var _ = errors.New
var _ = sort.Slice
