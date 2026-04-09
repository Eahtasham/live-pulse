package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Eahtasham/live-pulse/apps/api/internal/models"
)

// AuthService defines the interface the auth handler depends on.
type AuthService interface {
	FindOrCreateUser(email, name, avatarURL, provider string) (*models.User, error)
	RegisterUser(email, name, password string) (*models.User, error)
	LoginUser(email, password string) (*models.User, error)
	GenerateJWT(userID, email string) (string, error)
}

type callbackRequest struct {
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Provider  string `json:"provider"`
}

type registerRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string `json:"token"`
}

type AuthHandler struct {
	svc AuthService
}

func NewAuthHandler(svc AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Callback handles POST /v1/auth/callback.
// It finds or creates a user and returns a signed JWT.
func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	var req callbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "bad_request",
			"message": "invalid request body",
		})
		return
	}

	if req.Email == "" || req.Provider == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "bad_request",
			"message": "email and provider are required",
		})
		return
	}

	user, err := h.svc.FindOrCreateUser(req.Email, req.Name, req.AvatarURL, req.Provider)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "internal",
			"message": "failed to process user",
		})
		return
	}

	token, err := h.svc.GenerateJWT(user.ID.String(), user.Email)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "internal",
			"message": "failed to generate token",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(authResponse{Token: token})
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "bad_request",
			"message": "invalid request body",
		})
		return
	}

	if req.Email == "" || req.Password == "" || req.Name == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "bad_request",
			"message": "email, name, and password are required",
		})
		return
	}

	user, err := h.svc.RegisterUser(req.Email, req.Name, req.Password)
	if err != nil {
		if err.Error() == "email already registered" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "conflict",
				"message": "email already registered",
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "internal",
			"message": "failed to register user",
		})
		return
	}

	token, err := h.svc.GenerateJWT(user.ID.String(), user.Email)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "internal",
			"message": "failed to generate token",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(authResponse{Token: token})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "bad_request",
			"message": "invalid request body",
		})
		return
	}

	if req.Email == "" || req.Password == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "bad_request",
			"message": "email and password are required",
		})
		return
	}

	user, err := h.svc.LoginUser(req.Email, req.Password)
	if err != nil {
		if err.Error() == "invalid email or password" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "unauthorized",
				"message": "invalid email or password",
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "internal",
			"message": "failed to login",
		})
		return
	}

	token, err := h.svc.GenerateJWT(user.ID.String(), user.Email)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "internal",
			"message": "failed to generate token",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(authResponse{Token: token})
}
