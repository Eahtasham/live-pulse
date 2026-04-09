package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/Eahtasham/live-pulse/apps/api/internal/middleware"
	"github.com/Eahtasham/live-pulse/apps/api/internal/models"
	"github.com/Eahtasham/live-pulse/apps/api/internal/router"
)

const testSecret = "test-secret-key-for-unit-tests"

// --- mock auth service ------------------------------------------------

type mockAuthService struct {
	users map[string]*models.User // keyed by email
}

func newMockAuthService() *mockAuthService {
	return &mockAuthService{users: make(map[string]*models.User)}
}

func (m *mockAuthService) FindOrCreateUser(email, name, avatarURL, provider string) (*models.User, error) {
	if u, ok := m.users[email]; ok {
		return u, nil
	}
	u := &models.User{
		ID:       uuid.New(),
		Email:    email,
		Name:     strPtr(name),
		Provider: provider,
	}
	m.users[email] = u
	return u, nil
}

func (m *mockAuthService) RegisterUser(email, name, password string) (*models.User, error) {
	if _, ok := m.users[email]; ok {
		return nil, fmt.Errorf("email already registered")
	}
	u := &models.User{
		ID:       uuid.New(),
		Email:    email,
		Name:     strPtr(name),
		Provider: "email",
	}
	m.users[email] = u
	return u, nil
}

func (m *mockAuthService) LoginUser(email, password string) (*models.User, error) {
	u, ok := m.users[email]
	if !ok || u.Provider != "email" {
		return nil, fmt.Errorf("invalid email or password")
	}
	// Mock accepts any password for existing email-provider users
	return u, nil
}

func (m *mockAuthService) GenerateJWT(userID, email string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"iat":     time.Now().Unix(),
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(testSecret))
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// --- helpers -----------------------------------------------------------

func setupRouter(t *testing.T) (*mockAuthService, http.Handler) {
	t.Helper()
	svc := newMockAuthService()
	r := router.New(time.Now(), svc, testSecret)
	return svc, r
}

func generateTestJWT(t *testing.T, userID, email, secret string, expiry time.Duration) string {
	t.Helper()
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"iat":     time.Now().Unix(),
		"exp":     time.Now().Add(expiry).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign test jwt: %v", err)
	}
	return signed
}

// --- acceptance tests --------------------------------------------------

// POST /v1/sessions without Authorization header → 401
func TestCreateSession_NoAuth_Returns401(t *testing.T) {
	_, r := setupRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/v1/sessions", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}

	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["error"] != "unauthorized" {
		t.Errorf("expected error=unauthorized, got %q", body["error"])
	}
}

// POST /v1/sessions with valid JWT → 200
func TestCreateSession_ValidJWT_Returns200(t *testing.T) {
	_, r := setupRouter(t)

	token := generateTestJWT(t, "550e8400-e29b-41d4-a716-446655440000", "user@example.com", testSecret, 24*time.Hour)
	body := bytes.NewBufferString(`{"title": "Test Session"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

// POST /v1/sessions with expired JWT → 401 (not 500)
func TestCreateSession_ExpiredJWT_Returns401(t *testing.T) {
	_, r := setupRouter(t)

	token := generateTestJWT(t, "550e8400-e29b-41d4-a716-446655440000", "user@example.com", testSecret, -1*time.Hour)
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// POST /v1/sessions with malformed JWT → 401
func TestCreateSession_MalformedJWT_Returns401(t *testing.T) {
	_, r := setupRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/v1/sessions", nil)
	req.Header.Set("Authorization", "Bearer not-a-valid-jwt")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// Invalid JWT signature (wrong secret) → 401
func TestCreateSession_WrongSecret_Returns401(t *testing.T) {
	_, r := setupRouter(t)

	token := generateTestJWT(t, "550e8400-e29b-41d4-a716-446655440000", "user@example.com", "wrong-secret", 24*time.Hour)
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// GET /v1/sessions/:code without JWT → 200 (public)
func TestGetSession_NoAuth_Returns200(t *testing.T) {
	_, r := setupRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/ABC123", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

// POST /v1/auth/callback creates user and returns JWT
func TestAuthCallback_CreatesUserAndReturnsJWT(t *testing.T) {
	svc, r := setupRouter(t)

	payload := `{"email":"alice@example.com","name":"Alice","avatar_url":"https://avatar.example.com/alice.png","provider":"google"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/callback", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp["token"] == "" {
		t.Fatal("expected non-empty token in response")
	}

	// Verify user was stored in mock
	if _, ok := svc.users["alice@example.com"]; !ok {
		t.Error("expected user to be stored in service")
	}

	// Verify the JWT contains correct claims
	token, err := jwt.Parse(resp["token"], func(t *jwt.Token) (interface{}, error) {
		return []byte(testSecret), nil
	})
	if err != nil {
		t.Fatalf("failed to parse returned JWT: %v", err)
	}
	claims := token.Claims.(jwt.MapClaims)
	if claims["email"] != "alice@example.com" {
		t.Errorf("expected email=alice@example.com, got %v", claims["email"])
	}
	if claims["user_id"] == nil || claims["user_id"] == "" {
		t.Error("expected non-empty user_id in JWT claims")
	}
}

// Calling callback again with same email does NOT create duplicate user
func TestAuthCallback_NoDuplicateUser(t *testing.T) {
	svc, r := setupRouter(t)

	payload := `{"email":"bob@example.com","name":"Bob","avatar_url":"","provider":"github"}`

	// First call
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/callback", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("first callback: expected 200, got %d", rec.Code)
	}
	firstID := svc.users["bob@example.com"].ID

	// Second call — same email
	req = httptest.NewRequest(http.MethodPost, "/v1/auth/callback", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("second callback: expected 200, got %d", rec.Code)
	}

	if svc.users["bob@example.com"].ID != firstID {
		t.Error("expected same user ID on second callback, got a different one")
	}
	if len(svc.users) != 1 {
		t.Errorf("expected 1 user in mock, got %d", len(svc.users))
	}
}

// JWT expiry is set correctly (24h default)
func TestJWTExpiry(t *testing.T) {
	svc := newMockAuthService()
	token, err := svc.GenerateJWT("test-id", "test@example.com")
	if err != nil {
		t.Fatalf("failed to generate jwt: %v", err)
	}

	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte(testSecret), nil
	})
	if err != nil {
		t.Fatalf("failed to parse jwt: %v", err)
	}

	claims := parsed.Claims.(jwt.MapClaims)
	exp, _ := claims.GetExpirationTime()
	iat, _ := claims.GetIssuedAt()

	diff := exp.Sub(iat.Time)
	if diff < 23*time.Hour || diff > 25*time.Hour {
		t.Errorf("expected ~24h expiry, got %v", diff)
	}
}

// Context values are accessible after middleware
func TestMiddleware_SetsContextValues(t *testing.T) {
	handler := middleware.JWTAuth(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.UserIDFromContext(r.Context())
		email := middleware.UserEmailFromContext(r.Context())

		if userID != "test-user-id" {
			t.Errorf("expected user_id=test-user-id, got %q", userID)
		}
		if email != "test@example.com" {
			t.Errorf("expected email=test@example.com, got %q", email)
		}
		w.WriteHeader(http.StatusOK)
	}))

	token := generateTestJWT(t, "test-user-id", "test@example.com", testSecret, time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

// Uniform error message — no information leakage
func TestAuthErrors_UniformMessage(t *testing.T) {
	_, r := setupRouter(t)

	cases := []struct {
		name  string
		token string
	}{
		{"expired", generateTestJWT(t, "id", "e@e.com", testSecret, -time.Hour)},
		{"wrong_secret", generateTestJWT(t, "id", "e@e.com", "wrong", time.Hour)},
		{"malformed", "not.a.jwt"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/sessions", nil)
			req.Header.Set("Authorization", "Bearer "+tc.token)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			var body map[string]string
			json.NewDecoder(rec.Body).Decode(&body)

			if body["message"] != "invalid or expired token" {
				t.Errorf("[%s] expected uniform error message, got %q", tc.name, body["message"])
			}
		})
	}
}

// Ensure fmt imported (used transitively for error formatting check if needed)
var _ = fmt.Sprintf

// POST /v1/auth/register — success
func TestRegister_Success(t *testing.T) {
	svc, r := setupRouter(t)

	payload := `{"email":"newuser@example.com","name":"New User","password":"securepass123"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["token"] == "" {
		t.Fatal("expected non-empty token")
	}

	if _, ok := svc.users["newuser@example.com"]; !ok {
		t.Error("expected user to be stored")
	}
}

// POST /v1/auth/register — duplicate email → 409
func TestRegister_DuplicateEmail_Returns409(t *testing.T) {
	_, r := setupRouter(t)

	payload := `{"email":"dup@example.com","name":"First","password":"pass12345678"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("first register: expected 201, got %d", rec.Code)
	}

	// Second register with same email
	req = httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

// POST /v1/auth/register — missing fields → 400
func TestRegister_MissingFields_Returns400(t *testing.T) {
	_, r := setupRouter(t)

	payload := `{"email":"test@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

// POST /v1/auth/login — success
func TestLogin_Success(t *testing.T) {
	_, r := setupRouter(t)

	// Register first
	regPayload := `{"email":"login@example.com","name":"Login User","password":"mypassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewBufferString(regPayload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d", rec.Code)
	}

	// Login
	loginPayload := `{"email":"login@example.com","password":"mypassword123"}`
	req = httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(loginPayload))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["token"] == "" {
		t.Fatal("expected non-empty token")
	}
}

// POST /v1/auth/login — wrong credentials → 401
func TestLogin_WrongCredentials_Returns401(t *testing.T) {
	_, r := setupRouter(t)

	payload := `{"email":"nonexistent@example.com","password":"wrongpass"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// POST /v1/auth/login — missing fields → 400
func TestLogin_MissingFields_Returns400(t *testing.T) {
	_, r := setupRouter(t)

	payload := `{"email":"test@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
