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

// --- mock QA service ---------------------------------------------------

type mockQAService struct {
	mu       sync.Mutex
	sessions map[string]*models.Session
	entries  map[uuid.UUID]*models.QAEntry
	votes    map[string]*models.QAVote // entryID:voterUID → vote
}

func newMockQAService() *mockQAService {
	return &mockQAService{
		sessions: make(map[string]*models.Session),
		entries:  make(map[uuid.UUID]*models.QAEntry),
		votes:    make(map[string]*models.QAVote),
	}
}

func (m *mockQAService) addSession(code string, hostID uuid.UUID, status string) *models.Session {
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

func (m *mockQAService) addEntry(sessionCode, authorUID, entryType, body string) *models.QAEntry {
	m.mu.Lock()
	defer m.mu.Unlock()

	session := m.sessions[sessionCode]
	if session == nil {
		return nil
	}

	entry := &models.QAEntry{
		ID:        uuid.New(),
		SessionID: session.ID,
		AuthorUID: authorUID,
		EntryType: entryType,
		Body:      body,
		Score:     0,
		Status:    "visible",
		IsHidden:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.entries[entry.ID] = entry
	return entry
}

func (m *mockQAService) CreateEntry(sessionCode, authorUID, entryType, body string) (*models.QAEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate entry type
	if entryType != "question" && entryType != "comment" {
		return nil, service.ErrInvalidRequest
	}

	// Validate body
	if body == "" {
		return nil, service.ErrInvalidRequest
	}
	if len(body) > 2000 {
		return nil, service.ErrInvalidRequest
	}

	// Find session
	session, ok := m.sessions[sessionCode]
	if !ok {
		return nil, service.ErrSessionNotFound
	}

	// Check if session is archived
	if session.Status == "archived" {
		return nil, service.ErrInvalidRequest
	}

	entry := &models.QAEntry{
		ID:        uuid.New(),
		SessionID: session.ID,
		AuthorUID: authorUID,
		EntryType: entryType,
		Body:      body,
		Score:     0,
		Status:    "visible",
		IsHidden:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.entries[entry.ID] = entry
	return entry, nil
}

func (m *mockQAService) ListEntries(ctx context.Context, sessionCode, cursor string, limit int, audienceUID string) ([]service.QAEntryWithVote, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[sessionCode]
	if !ok {
		return nil, "", service.ErrSessionNotFound
	}

	var results []service.QAEntryWithVote
	for _, e := range m.entries {
		if e.SessionID == session.ID && !e.IsHidden {
			entryWithVote := service.QAEntryWithVote{QAEntry: *e}
			// Include user's vote if audienceUID provided
			if audienceUID != "" {
				voteKey := e.ID.String() + ":" + audienceUID
				if vote, exists := m.votes[voteKey]; exists {
					entryWithVote.UserVote = &vote.VoteValue
				}
			}
			results = append(results, entryWithVote)
		}
	}

	// Simple sort by score DESC, then created_at ASC
	// (simplified for mock)
	return results, "", nil
}

func (m *mockQAService) ModerateEntry(sessionCode string, entryID, hostID uuid.UUID, status string, isHidden *bool) (*models.QAEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[sessionCode]
	if !ok {
		return nil, service.ErrSessionNotFound
	}

	// Verify host ownership
	if session.HostID == nil || *session.HostID != hostID {
		return nil, service.ErrNotSessionHost
	}

	entry, ok := m.entries[entryID]
	if !ok || entry.SessionID != session.ID {
		return nil, service.ErrQAEntryNotFound
	}

	if status != "" {
		validStatuses := map[string]bool{
			"visible":  true,
			"answered": true,
			"pinned":   true,
			"archived": true,
		}
		if !validStatuses[status] {
			return nil, service.ErrInvalidRequest
		}
		entry.Status = status
	}

	if isHidden != nil {
		entry.IsHidden = *isHidden
	}

	entry.UpdatedAt = time.Now()
	return entry, nil
}

func (m *mockQAService) GetEntry(sessionCode string, entryID uuid.UUID) (*models.QAEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[sessionCode]
	if !ok {
		return nil, service.ErrSessionNotFound
	}

	entry, ok := m.entries[entryID]
	if !ok || entry.SessionID != session.ID {
		return nil, service.ErrQAEntryNotFound
	}

	return entry, nil
}

func (m *mockQAService) GetSessionByCode(code string) (*models.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[code]
	if !ok {
		return nil, service.ErrSessionNotFound
	}
	return session, nil
}

// --- mock QA vote service ---------------------------------------------------

type mockQAVoteService struct {
	mu    sync.Mutex
	qaSvc *mockQAService
	votes map[string]*models.QAVote // entryID:voterUID → vote
}

func newMockQAVoteService(qaSvc *mockQAService) *mockQAVoteService {
	return &mockQAVoteService{
		qaSvc: qaSvc,
		votes: make(map[string]*models.QAVote),
	}
}

func (m *mockQAVoteService) CastVote(sessionCode string, entryID uuid.UUID, voterUID string, value int16) (*models.QAVote, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate vote value
	if value != 1 && value != -1 {
		return nil, service.ErrInvalidVoteValue
	}

	// Check session exists
	session, ok := m.qaSvc.sessions[sessionCode]
	if !ok {
		return nil, service.ErrSessionNotFound
	}

	// Find entry
	entry, ok := m.qaSvc.entries[entryID]
	if !ok || entry.SessionID != session.ID {
		return nil, service.ErrQAEntryNotFound
	}

	// Cannot vote on comments
	if entry.EntryType == "comment" {
		return nil, service.ErrQAEntryIsComment
	}

	// Cannot vote on hidden or archived entries
	if entry.IsHidden {
		return nil, service.ErrQAEntryNotVisible
	}
	if entry.Status == "archived" {
		return nil, service.ErrQAEntryArchived
	}

	voteKey := fmt.Sprintf("%s:%s", entryID.String(), voterUID)
	existingVote, exists := m.votes[voteKey]

	if exists {
		// Toggle behavior
		if existingVote.VoteValue == int16(value) {
			// Same value - remove vote
			delete(m.votes, voteKey)
			// Update score
			entry.Score -= int(value)
			return nil, nil
		} else {
			// Different vote value - update the vote
			// Calculate score change BEFORE updating vote value
			scoreChange := int(value) - int(existingVote.VoteValue)
			existingVote.VoteValue = int16(value)
			// Update score
			entry.Score += scoreChange
			return existingVote, nil
		}
	}

	// Create new vote
	vote := &models.QAVote{
		ID:        uuid.New(),
		QAEntryID: entryID,
		VoterUID:  voterUID,
		VoteValue: int16(value),
		CreatedAt: time.Now(),
	}
	m.votes[voteKey] = vote
	entry.Score += int(value)

	return vote, nil
}

func (m *mockQAVoteService) GetEntry(sessionCode string, entryID uuid.UUID) (*models.QAEntry, error) {
	return m.qaSvc.GetEntry(sessionCode, entryID)
}

// --- helpers ---------------------------------------------------------------

func setupRouterWithQA(qaSvc *mockQAService, qaVoteSvc *mockQAVoteService) http.Handler {
	return router.New(
		time.Now(),
		nil, // authSvc
		nil, // sessionSvc
		nil, // pollSvc
		nil, // voteSvc
		qaSvc,
		qaVoteSvc,
		testSecret,
		[]string{"http://localhost:3000"},
	)
}

// --- tests -----------------------------------------------------------------

func TestQACreate_Success(t *testing.T) {
	qaSvc := newMockQAService()
	hostID := uuid.New()
	qaSvc.addSession("TEST01", hostID, "active")

	router := setupRouterWithQA(qaSvc, nil)

	reqBody := map[string]string{
		"entry_type": "question",
		"body":       "What is Big-O notation?",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/TEST01/qa", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Audience-UID", "audience-123")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["entry_type"] != "question" {
		t.Errorf("expected entry_type 'question', got %v", resp["entry_type"])
	}
	if resp["body"] != "What is Big-O notation?" {
		t.Errorf("expected body 'What is Big-O notation?', got %v", resp["body"])
	}
	if resp["author_uid"] != "audience-123" {
		t.Errorf("expected author_uid 'audience-123', got %v", resp["author_uid"])
	}
}

func TestQACreate_CommentSuccess(t *testing.T) {
	qaSvc := newMockQAService()
	hostID := uuid.New()
	qaSvc.addSession("TEST01", hostID, "active")

	router := setupRouterWithQA(qaSvc, nil)

	reqBody := map[string]string{
		"entry_type": "comment",
		"body":       "Great lecture!",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/TEST01/qa", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Audience-UID", "audience-123")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["entry_type"] != "comment" {
		t.Errorf("expected entry_type 'comment', got %v", resp["entry_type"])
	}
}

func TestQACreate_EmptyBody(t *testing.T) {
	qaSvc := newMockQAService()
	hostID := uuid.New()
	qaSvc.addSession("TEST01", hostID, "active")

	router := setupRouterWithQA(qaSvc, nil)

	reqBody := map[string]string{
		"entry_type": "question",
		"body":       "",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/TEST01/qa", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Audience-UID", "audience-123")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestQACreate_BodyTooLong(t *testing.T) {
	qaSvc := newMockQAService()
	hostID := uuid.New()
	qaSvc.addSession("TEST01", hostID, "active")

	router := setupRouterWithQA(qaSvc, nil)

	reqBody := map[string]string{
		"entry_type": "question",
		"body":       string(make([]byte, 2001)),
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/TEST01/qa", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Audience-UID", "audience-123")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestQACreate_InvalidEntryType(t *testing.T) {
	qaSvc := newMockQAService()
	hostID := uuid.New()
	qaSvc.addSession("TEST01", hostID, "active")

	router := setupRouterWithQA(qaSvc, nil)

	reqBody := map[string]string{
		"entry_type": "invalid",
		"body":       "Some text",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/TEST01/qa", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Audience-UID", "audience-123")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestQACreate_SessionNotFound(t *testing.T) {
	qaSvc := newMockQAService()

	router := setupRouterWithQA(qaSvc, nil)

	reqBody := map[string]string{
		"entry_type": "question",
		"body":       "What is this?",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/INVALID/qa", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Audience-UID", "audience-123")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestQACreate_ArchivedSession(t *testing.T) {
	qaSvc := newMockQAService()
	hostID := uuid.New()
	qaSvc.addSession("TEST01", hostID, "archived")

	router := setupRouterWithQA(qaSvc, nil)

	reqBody := map[string]string{
		"entry_type": "question",
		"body":       "What is this?",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/TEST01/qa", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Audience-UID", "audience-123")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestQACreate_MissingAudienceUID(t *testing.T) {
	qaSvc := newMockQAService()
	hostID := uuid.New()
	qaSvc.addSession("TEST01", hostID, "active")

	router := setupRouterWithQA(qaSvc, nil)

	reqBody := map[string]string{
		"entry_type": "question",
		"body":       "What is this?",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/TEST01/qa", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No X-Audience-UID header

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestQAList_Success(t *testing.T) {
	qaSvc := newMockQAService()
	hostID := uuid.New()
	qaSvc.addSession("TEST01", hostID, "active")
	qaSvc.addEntry("TEST01", "audience-1", "question", "Question 1")
	qaSvc.addEntry("TEST01", "audience-2", "comment", "Comment 1")

	router := setupRouterWithQA(qaSvc, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/TEST01/qa", nil)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	entries, ok := resp["entries"].([]interface{})
	if !ok {
		t.Fatalf("expected entries array, got %T", resp["entries"])
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

func TestQAList_HiddenEntriesNotReturned(t *testing.T) {
	qaSvc := newMockQAService()
	hostID := uuid.New()
	qaSvc.addSession("TEST01", hostID, "active")
	entry := qaSvc.addEntry("TEST01", "audience-1", "question", "Visible question")
	qaSvc.addEntry("TEST01", "audience-2", "question", "Hidden question")

	// Hide the second entry manually
	for _, e := range qaSvc.entries {
		if e.ID != entry.ID {
			e.IsHidden = true
		}
	}

	router := setupRouterWithQA(qaSvc, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/TEST01/qa", nil)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	entries, ok := resp["entries"].([]interface{})
	if !ok {
		t.Fatalf("expected entries array, got %T", resp["entries"])
	}

	// Only 1 visible entry
	if len(entries) != 1 {
		t.Errorf("expected 1 visible entry, got %d", len(entries))
	}
}

func TestQAList_SessionNotFound(t *testing.T) {
	qaSvc := newMockQAService()

	router := setupRouterWithQA(qaSvc, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/INVALID/qa", nil)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestQAModerate_MarkAnswered(t *testing.T) {
	qaSvc := newMockQAService()
	hostID := uuid.New()
	qaSvc.addSession("TEST01", hostID, "active")
	entry := qaSvc.addEntry("TEST01", "audience-1", "question", "Question?")

	qaVoteSvc := newMockQAVoteService(qaSvc)
	router := setupRouterWithQA(qaSvc, qaVoteSvc)

	// Generate JWT for host
	token := generateTestJWT(t, hostID.String(), "host@example.com", testSecret, 24*time.Hour)

	reqBody := map[string]string{
		"status": "answered",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/v1/sessions/TEST01/qa/%s", entry.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["status"] != "answered" {
		t.Errorf("expected status 'answered', got %v", resp["status"])
	}
}

func TestQAModerate_PinQuestion(t *testing.T) {
	qaSvc := newMockQAService()
	hostID := uuid.New()
	qaSvc.addSession("TEST01", hostID, "active")
	entry := qaSvc.addEntry("TEST01", "audience-1", "question", "Question?")

	qaVoteSvc := newMockQAVoteService(qaSvc)
	router := setupRouterWithQA(qaSvc, qaVoteSvc)

	token := generateTestJWT(t, hostID.String(), "host@example.com", testSecret, 24*time.Hour)

	reqBody := map[string]string{
		"status": "pinned",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/v1/sessions/TEST01/qa/%s", entry.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["status"] != "pinned" {
		t.Errorf("expected status 'pinned', got %v", resp["status"])
	}
}

func TestQAModerate_HideQuestion(t *testing.T) {
	qaSvc := newMockQAService()
	hostID := uuid.New()
	qaSvc.addSession("TEST01", hostID, "active")
	entry := qaSvc.addEntry("TEST01", "audience-1", "question", "Question?")
	// Mark as answered first
	entry.Status = "answered"

	qaVoteSvc := newMockQAVoteService(qaSvc)
	router := setupRouterWithQA(qaSvc, qaVoteSvc)

	token := generateTestJWT(t, hostID.String(), "host@example.com", testSecret, 24*time.Hour)

	hide := true
	reqBody := map[string]bool{
		"is_hidden": hide,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/v1/sessions/TEST01/qa/%s", entry.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["is_hidden"] != true {
		t.Errorf("expected is_hidden true, got %v", resp["is_hidden"])
	}
	// Status should remain answered
	if resp["status"] != "answered" {
		t.Errorf("expected status 'answered' (unchanged), got %v", resp["status"])
	}
}

func TestQAModerate_NonHostForbidden(t *testing.T) {
	qaSvc := newMockQAService()
	hostID := uuid.New()
	nonHostID := uuid.New()
	qaSvc.addSession("TEST01", hostID, "active")
	entry := qaSvc.addEntry("TEST01", "audience-1", "question", "Question?")

	qaVoteSvc := newMockQAVoteService(qaSvc)
	router := setupRouterWithQA(qaSvc, qaVoteSvc)

	token := generateTestJWT(t, nonHostID.String(), "user@example.com", testSecret, 24*time.Hour)

	reqBody := map[string]string{
		"status": "answered",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/v1/sessions/TEST01/qa/%s", entry.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestQAModerate_EntryNotFound(t *testing.T) {
	qaSvc := newMockQAService()
	hostID := uuid.New()
	qaSvc.addSession("TEST01", hostID, "active")

	qaVoteSvc := newMockQAVoteService(qaSvc)
	router := setupRouterWithQA(qaSvc, qaVoteSvc)

	token := generateTestJWT(t, hostID.String(), "host@example.com", testSecret, 24*time.Hour)

	reqBody := map[string]string{
		"status": "answered",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/v1/sessions/TEST01/qa/%s", uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestQAModerate_InvalidStatus(t *testing.T) {
	qaSvc := newMockQAService()
	hostID := uuid.New()
	qaSvc.addSession("TEST01", hostID, "active")
	entry := qaSvc.addEntry("TEST01", "audience-1", "question", "Question?")

	qaVoteSvc := newMockQAVoteService(qaSvc)
	router := setupRouterWithQA(qaSvc, qaVoteSvc)

	token := generateTestJWT(t, hostID.String(), "host@example.com", testSecret, 24*time.Hour)

	reqBody := map[string]string{
		"status": "invalid_status",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/v1/sessions/TEST01/qa/%s", entry.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestQAModerate_NoAuth(t *testing.T) {
	qaSvc := newMockQAService()
	hostID := uuid.New()
	qaSvc.addSession("TEST01", hostID, "active")
	entry := qaSvc.addEntry("TEST01", "audience-1", "question", "Question?")

	qaVoteSvc := newMockQAVoteService(qaSvc)
	router := setupRouterWithQA(qaSvc, qaVoteSvc)

	reqBody := map[string]string{
		"status": "answered",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/v1/sessions/TEST01/qa/%s", entry.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

// --- QA Vote Tests ---------------------------------------------------------

func TestQAVote_UpvoteSuccess(t *testing.T) {
	qaSvc := newMockQAService()
	hostID := uuid.New()
	qaSvc.addSession("TEST01", hostID, "active")
	entry := qaSvc.addEntry("TEST01", "audience-1", "question", "Question?")

	qaVoteSvc := newMockQAVoteService(qaSvc)
	router := setupRouterWithQA(qaSvc, qaVoteSvc)

	reqBody := map[string]interface{}{
		"audience_uid": "audience-2",
		"value":        1,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/TEST01/qa/%s/vote", entry.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["action"] != "voted" {
		t.Errorf("expected action 'voted', got %v", resp["action"])
	}
	if resp["vote_value"] != float64(1) {
		t.Errorf("expected vote_value 1, got %v", resp["vote_value"])
	}

	// Check score updated
	if entry.Score != 1 {
		t.Errorf("expected entry score 1, got %d", entry.Score)
	}
}

func TestQAVote_DownvoteSuccess(t *testing.T) {
	qaSvc := newMockQAService()
	hostID := uuid.New()
	qaSvc.addSession("TEST01", hostID, "active")
	entry := qaSvc.addEntry("TEST01", "audience-1", "question", "Question?")

	qaVoteSvc := newMockQAVoteService(qaSvc)
	router := setupRouterWithQA(qaSvc, qaVoteSvc)

	reqBody := map[string]interface{}{
		"audience_uid": "audience-2",
		"value":        -1,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/TEST01/qa/%s/vote", entry.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["action"] != "voted" {
		t.Errorf("expected action 'voted', got %v", resp["action"])
	}
	if resp["vote_value"] != float64(-1) {
		t.Errorf("expected vote_value -1, got %v", resp["vote_value"])
	}

	// Check score updated
	if entry.Score != -1 {
		t.Errorf("expected entry score -1, got %d", entry.Score)
	}
}

func TestQAVote_ToggleRemoveVote(t *testing.T) {
	qaSvc := newMockQAService()
	hostID := uuid.New()
	qaSvc.addSession("TEST01", hostID, "active")
	entry := qaSvc.addEntry("TEST01", "audience-1", "question", "Question?")

	qaVoteSvc := newMockQAVoteService(qaSvc)
	router := setupRouterWithQA(qaSvc, qaVoteSvc)

	audienceUID := "audience-2"

	// First upvote
	reqBody := map[string]interface{}{
		"audience_uid": audienceUID,
		"value":        1,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/TEST01/qa/%s/vote", entry.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("first vote expected 200, got %d", rr.Code)
	}

	if entry.Score != 1 {
		t.Errorf("expected score 1 after first vote, got %d", entry.Score)
	}

	// Second upvote (same value) - should remove
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/TEST01/qa/%s/vote", entry.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("second vote expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["action"] != "removed" {
		t.Errorf("expected action 'removed', got %v", resp["action"])
	}

	// Score should be back to 0
	if entry.Score != 0 {
		t.Errorf("expected score 0 after toggle, got %d", entry.Score)
	}
}

func TestQAVote_ChangeVote(t *testing.T) {
	qaSvc := newMockQAService()
	hostID := uuid.New()
	qaSvc.addSession("TEST01", hostID, "active")
	entry := qaSvc.addEntry("TEST01", "audience-1", "question", "Question?")

	qaVoteSvc := newMockQAVoteService(qaSvc)
	router := setupRouterWithQA(qaSvc, qaVoteSvc)

	audienceUID := "audience-2"

	// First upvote
	reqBody := map[string]interface{}{
		"audience_uid": audienceUID,
		"value":        1,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/TEST01/qa/%s/vote", entry.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("first vote expected 200, got %d", rr.Code)
	}

	if entry.Score != 1 {
		t.Errorf("expected score 1, got %d", entry.Score)
	}

	// Change to downvote
	reqBody = map[string]interface{}{
		"audience_uid": audienceUID,
		"value":        -1,
	}
	body, _ = json.Marshal(reqBody)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/TEST01/qa/%s/vote", entry.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("second vote expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["action"] != "voted" {
		t.Errorf("expected action 'voted', got %v", resp["action"])
	}
	if resp["vote_value"] != float64(-1) {
		t.Errorf("expected vote_value -1, got %v", resp["vote_value"])
	}

	// Score should be -1 (removed +1, added -1)
	if entry.Score != -1 {
		t.Errorf("expected score -1 after change, got %d", entry.Score)
	}
}

func TestQAVote_MultipleVoters(t *testing.T) {
	qaSvc := newMockQAService()
	hostID := uuid.New()
	qaSvc.addSession("TEST01", hostID, "active")
	entry := qaSvc.addEntry("TEST01", "audience-1", "question", "Question?")

	qaVoteSvc := newMockQAVoteService(qaSvc)
	router := setupRouterWithQA(qaSvc, qaVoteSvc)

	// 3 upvotes from different users
	for i := 0; i < 3; i++ {
		reqBody := map[string]interface{}{
			"audience_uid": fmt.Sprintf("audience-%d", i),
			"value":        1,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/TEST01/qa/%s/vote", entry.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("vote %d expected 200, got %d", i, rr.Code)
		}
	}

	// 1 downvote
	reqBody := map[string]interface{}{
		"audience_uid": "audience-down",
		"value":        -1,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/TEST01/qa/%s/vote", entry.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("downvote expected 200, got %d", rr.Code)
	}

	// Score should be 3 - 1 = 2
	if entry.Score != 2 {
		t.Errorf("expected score 2 (3 up - 1 down), got %d", entry.Score)
	}
}

func TestQAVote_CannotVoteOnComment(t *testing.T) {
	qaSvc := newMockQAService()
	hostID := uuid.New()
	qaSvc.addSession("TEST01", hostID, "active")
	entry := qaSvc.addEntry("TEST01", "audience-1", "comment", "A comment")

	qaVoteSvc := newMockQAVoteService(qaSvc)
	router := setupRouterWithQA(qaSvc, qaVoteSvc)

	reqBody := map[string]interface{}{
		"audience_uid": "audience-2",
		"value":        1,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/TEST01/qa/%s/vote", entry.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestQAVote_CannotVoteOnHiddenEntry(t *testing.T) {
	qaSvc := newMockQAService()
	hostID := uuid.New()
	qaSvc.addSession("TEST01", hostID, "active")
	entry := qaSvc.addEntry("TEST01", "audience-1", "question", "Question?")
	entry.IsHidden = true

	qaVoteSvc := newMockQAVoteService(qaSvc)
	router := setupRouterWithQA(qaSvc, qaVoteSvc)

	reqBody := map[string]interface{}{
		"audience_uid": "audience-2",
		"value":        1,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/TEST01/qa/%s/vote", entry.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestQAVote_CannotVoteOnArchivedEntry(t *testing.T) {
	qaSvc := newMockQAService()
	hostID := uuid.New()
	qaSvc.addSession("TEST01", hostID, "active")
	entry := qaSvc.addEntry("TEST01", "audience-1", "question", "Question?")
	entry.Status = "archived"

	qaVoteSvc := newMockQAVoteService(qaSvc)
	router := setupRouterWithQA(qaSvc, qaVoteSvc)

	reqBody := map[string]interface{}{
		"audience_uid": "audience-2",
		"value":        1,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/TEST01/qa/%s/vote", entry.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestQAVote_InvalidValue(t *testing.T) {
	qaSvc := newMockQAService()
	hostID := uuid.New()
	qaSvc.addSession("TEST01", hostID, "active")
	entry := qaSvc.addEntry("TEST01", "audience-1", "question", "Question?")

	qaVoteSvc := newMockQAVoteService(qaSvc)
	router := setupRouterWithQA(qaSvc, qaVoteSvc)

	reqBody := map[string]interface{}{
		"audience_uid": "audience-2",
		"value":        5, // Invalid value
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/TEST01/qa/%s/vote", entry.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestQAVote_EntryNotFound(t *testing.T) {
	qaSvc := newMockQAService()
	hostID := uuid.New()
	qaSvc.addSession("TEST01", hostID, "active")

	qaVoteSvc := newMockQAVoteService(qaSvc)
	router := setupRouterWithQA(qaSvc, qaVoteSvc)

	reqBody := map[string]interface{}{
		"audience_uid": "audience-2",
		"value":        1,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/TEST01/qa/%s/vote", uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestQAVote_SessionNotFound(t *testing.T) {
	qaSvc := newMockQAService()

	qaVoteSvc := newMockQAVoteService(qaSvc)
	router := setupRouterWithQA(qaSvc, qaVoteSvc)

	reqBody := map[string]interface{}{
		"audience_uid": "audience-2",
		"value":        1,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/sessions/INVALID/qa/%s/vote", uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}
