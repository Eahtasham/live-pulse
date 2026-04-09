package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Eahtasham/live-pulse/apps/api/internal/models"
	"github.com/Eahtasham/live-pulse/apps/api/internal/router"
)

// --- mock session service ------------------------------------------------

type mockSessionService struct {
	mu       sync.Mutex
	sessions map[string]*models.Session // keyed by code
	byHost   map[string][]string        // hostID → []code
	audience map[string]string          // "code:clientID" → uid
	uidKeys  map[string]time.Duration   // "audience:code:uid" → TTL
}

func newMockSessionService() *mockSessionService {
	return &mockSessionService{
		sessions: make(map[string]*models.Session),
		byHost:   make(map[string][]string),
		audience: make(map[string]string),
		uidKeys:  make(map[string]time.Duration),
	}
}

func (m *mockSessionService) CreateSession(hostID uuid.UUID, title string) (*models.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate a simple code for testing
	code := generateMockCode(m.sessions)

	session := &models.Session{
		ID:        uuid.New(),
		HostID:    &hostID,
		Code:      code,
		Title:     title,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.sessions[code] = session
	m.byHost[hostID.String()] = append(m.byHost[hostID.String()], code)
	return session, nil
}

func (m *mockSessionService) ListSessionsByHost(hostID uuid.UUID) ([]models.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	codes := m.byHost[hostID.String()]
	var result []models.Session
	// Return in reverse order (newest first)
	for i := len(codes) - 1; i >= 0; i-- {
		if s, ok := m.sessions[codes[i]]; ok {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *mockSessionService) GetSessionByCode(code string) (*models.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.sessions[code]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}
	return s, nil
}

func (m *mockSessionService) JoinSession(ctx context.Context, code, clientID string) (string, *models.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.sessions[code]
	if !ok {
		return "", nil, fmt.Errorf("session not found")
	}

	// Check for existing mapping
	if clientID != "" {
		key := code + ":" + clientID
		if uid, exists := m.audience[key]; exists {
			return uid, s, nil
		}
	}

	uid := uuid.New().String()

	// Store audience:{code}:{uid} key
	uidKey := "audience:" + code + ":" + uid
	m.uidKeys[uidKey] = 24 * time.Hour

	if clientID != "" {
		m.audience[code+":"+clientID] = uid
	}

	return uid, s, nil
}

func generateMockCode(existing map[string]*models.Session) string {
	charset := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	// Simple incrementing code for testing determinism
	n := len(existing)
	code := make([]byte, 6)
	for i := 5; i >= 0; i-- {
		code[i] = charset[n%len(charset)]
		n /= len(charset)
	}
	return string(code)
}

// --- acceptance tests --------------------------------------------------

// POST /v1/sessions with {title} → returns {id, code, title, status: "active"}
func TestCreateSession_Success(t *testing.T) {
	_, _, r := setupRouterWithSession(t)

	token := generateTestJWT(t, uuid.New().String(), "host@example.com", testSecret, 24*time.Hour)
	body := bytes.NewBufferString(`{"title": "CS101 Lecture"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions", body)
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
	if resp["title"] != "CS101 Lecture" {
		t.Errorf("expected title=CS101 Lecture, got %v", resp["title"])
	}
	if resp["status"] != "active" {
		t.Errorf("expected status=active, got %v", resp["status"])
	}

	// Code is exactly 6 alphanumeric uppercase characters
	code, ok := resp["code"].(string)
	if !ok || len(code) != 6 {
		t.Fatalf("expected 6-char code, got %q", code)
	}
	matched, _ := regexp.MatchString(`^[A-Z0-9]{6}$`, code)
	if !matched {
		t.Errorf("code %q does not match ^[A-Z0-9]{6}$", code)
	}
}

// Creating a session without auth → 401
func TestCreateSession_NoAuth_Returns401_Session(t *testing.T) {
	_, _, r := setupRouterWithSession(t)

	body := bytes.NewBufferString(`{"title": "No Auth Session"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// Creating 100 sessions produces 100 unique codes
func TestCreateSession_UniqueCodesFor100(t *testing.T) {
	_, _, r := setupRouterWithSession(t)

	token := generateTestJWT(t, uuid.New().String(), "host@example.com", testSecret, 24*time.Hour)
	codes := make(map[string]bool)

	for i := 0; i < 100; i++ {
		body := bytes.NewBufferString(`{"title": "Session"}`)
		req := httptest.NewRequest(http.MethodPost, "/v1/sessions", body)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("session %d: expected 201, got %d", i, rec.Code)
		}

		var resp map[string]interface{}
		json.NewDecoder(rec.Body).Decode(&resp)
		code := resp["code"].(string)
		if codes[code] {
			t.Fatalf("duplicate code %q at session %d", code, i)
		}
		codes[code] = true
	}

	if len(codes) != 100 {
		t.Errorf("expected 100 unique codes, got %d", len(codes))
	}
}

// GET /v1/sessions/:code → returns session details (no auth needed)
func TestGetSessionByCode_Public(t *testing.T) {
	_, sessionSvc, r := setupRouterWithSession(t)

	// Create a session in the mock
	hostID := uuid.New()
	session, _ := sessionSvc.CreateSession(hostID, "Public Session")

	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+session.Code, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["title"] != "Public Session" {
		t.Errorf("expected title=Public Session, got %v", resp["title"])
	}
}

// GET /v1/sessions/XXXXXX (invalid code) → 404
func TestGetSessionByCode_NotFound(t *testing.T) {
	_, _, r := setupRouterWithSession(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/XXXXXX", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

// GET /v1/sessions with auth → returns list sorted by created_at DESC
func TestListSessions_WithAuth(t *testing.T) {
	_, sessionSvc, r := setupRouterWithSession(t)

	hostID := uuid.New()
	sessionSvc.CreateSession(hostID, "Session 1")
	time.Sleep(time.Millisecond) // ensure ordering
	sessionSvc.CreateSession(hostID, "Session 2")

	token := generateTestJWT(t, hostID.String(), "host@example.com", testSecret, 24*time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var sessions []map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&sessions)

	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	// Newest first
	if sessions[0]["title"] != "Session 2" {
		t.Errorf("expected first session to be Session 2, got %v", sessions[0]["title"])
	}
}

// GET /v1/sessions without auth → 401
func TestListSessions_NoAuth_Returns401(t *testing.T) {
	_, _, r := setupRouterWithSession(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/sessions", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// POST /v1/sessions/:code/join → returns {audience_uid} and creates Redis key
func TestJoinSession_Success(t *testing.T) {
	_, sessionSvc, r := setupRouterWithSession(t)

	hostID := uuid.New()
	session, _ := sessionSvc.CreateSession(hostID, "Join Test")

	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+session.Code+"/join", nil)
	req.Header.Set("X-Client-ID", "client-123")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)

	uid, ok := resp["audience_uid"].(string)
	if !ok || uid == "" {
		t.Fatal("expected non-empty audience_uid")
	}

	if resp["session_title"] != "Join Test" {
		t.Errorf("expected session_title=Join Test, got %v", resp["session_title"])
	}

	// Verify Redis key with TTL exists in mock
	uidKey := "audience:" + session.Code + ":" + uid
	ttl, exists := sessionSvc.uidKeys[uidKey]
	if !exists {
		t.Error("expected audience UID key in Redis")
	}
	if ttl != 24*time.Hour {
		t.Errorf("expected TTL=24h, got %v", ttl)
	}
}

// Calling join again with same client ID → returns SAME uid (idempotent)
func TestJoinSession_Idempotent(t *testing.T) {
	_, sessionSvc, r := setupRouterWithSession(t)

	hostID := uuid.New()
	session, _ := sessionSvc.CreateSession(hostID, "Idempotent Test")

	// First join
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+session.Code+"/join", nil)
	req.Header.Set("X-Client-ID", "client-abc")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var resp1 map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp1)
	uid1 := resp1["audience_uid"].(string)

	// Second join — same client ID
	req = httptest.NewRequest(http.MethodPost, "/v1/sessions/"+session.Code+"/join", nil)
	req.Header.Set("X-Client-ID", "client-abc")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var resp2 map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp2)
	uid2 := resp2["audience_uid"].(string)

	if uid1 != uid2 {
		t.Errorf("expected same uid on second join, got %q and %q", uid1, uid2)
	}
}

// Different client joining → gets DIFFERENT uid
func TestJoinSession_DifferentClients(t *testing.T) {
	_, sessionSvc, r := setupRouterWithSession(t)

	hostID := uuid.New()
	session, _ := sessionSvc.CreateSession(hostID, "Multi Client Test")

	// Client 1
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+session.Code+"/join", nil)
	req.Header.Set("X-Client-ID", "client-1")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var resp1 map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp1)

	// Client 2
	req = httptest.NewRequest(http.MethodPost, "/v1/sessions/"+session.Code+"/join", nil)
	req.Header.Set("X-Client-ID", "client-2")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var resp2 map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp2)

	if resp1["audience_uid"] == resp2["audience_uid"] {
		t.Error("expected different UIDs for different clients")
	}
}

// Join invalid session → 404
func TestJoinSession_NotFound(t *testing.T) {
	_, _, r := setupRouterWithSession(t)

	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/ZZZZZZ/join", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

// --- helper to set up router with both mocks ---

func setupRouterWithSession(t *testing.T) (*mockAuthService, *mockSessionService, http.Handler) {
	t.Helper()
	authSvc := newMockAuthService()
	sessionSvc := newMockSessionService()
	r := router.New(time.Now(), authSvc, sessionSvc, testSecret)
	return authSvc, sessionSvc, r
}
