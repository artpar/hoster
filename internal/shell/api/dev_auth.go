// Package api provides HTTP handlers for the Hoster API.
// This file implements dev mode authentication endpoints for local development.
// Following the plan: https://github.com/artpar/apigate/issues/33
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// DevAuthHandlers provides authentication endpoints for development mode.
// These endpoints mimic what APIGate would provide, allowing the frontend
// to work without requiring a real APIGate instance with JSON API auth.
//
// IMPORTANT: Only enabled when auth.mode=dev. Never use in production.
type DevAuthHandlers struct {
	logger    *slog.Logger
	sessions  map[string]*DevSession // Simple in-memory session store
	sessionMu sync.RWMutex
}

// DevSession represents an authenticated dev session.
type DevSession struct {
	UserID    string
	Email     string
	Name      string
	CreatedAt time.Time
}

// DevUser represents the user data returned by auth endpoints.
type DevUser struct {
	ID         string        `json:"id"`
	Email      string        `json:"email"`
	Name       string        `json:"name"`
	PlanID     string        `json:"plan_id"`
	PlanLimits DevPlanLimits `json:"plan_limits"`
}

// DevPlanLimits represents plan limits for dev mode.
type DevPlanLimits struct {
	MaxDeployments int `json:"max_deployments"`
	MaxCPUCores    int `json:"max_cpu_cores"`
	MaxMemoryMB    int `json:"max_memory_mb"`
	MaxDiskGB      int `json:"max_disk_gb"`
}

// LoginRequest represents the login request body.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterRequest represents the registration request body.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// NewDevAuthHandlers creates new dev auth handlers.
func NewDevAuthHandlers(logger *slog.Logger) *DevAuthHandlers {
	if logger == nil {
		logger = slog.Default()
	}
	return &DevAuthHandlers{
		logger:   logger,
		sessions: make(map[string]*DevSession),
	}
}

// LookupSession looks up a session by session ID.
// Returns nil if the session is not found.
// This is used by the auth middleware to get the actual user ID from the session.
func (h *DevAuthHandlers) LookupSession(sessionID string) *DevSession {
	h.sessionMu.RLock()
	defer h.sessionMu.RUnlock()
	return h.sessions[sessionID]
}

// RegisterRoutes registers dev auth routes on the router.
// Auth endpoints are under /api/auth/ to avoid conflicting with APIGate's reserved /auth path.
func (h *DevAuthHandlers) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/auth/login", h.handleLogin).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/auth/register", h.handleRegister).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/auth/me", h.handleMe).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/auth/logout", h.handleLogout).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/auth/forgot", h.handleForgotPassword).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/auth/reset", h.handleResetPassword).Methods("POST", "OPTIONS")
}

// handleLogin handles POST /auth/login.
// In dev mode, accepts any credentials and returns a Bearer token.
func (h *DevAuthHandlers) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		h.handleCORS(w)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email == "" {
		h.writeError(w, http.StatusBadRequest, "Email is required")
		return
	}

	// In dev mode, accept any credentials
	h.logger.Info("dev auth: login", "email", req.Email)

	// Create a session keyed by token
	token := "tok_" + randomString(32)
	h.sessionMu.Lock()
	h.sessions[token] = &DevSession{
		UserID:    "dev-user-" + randomString(8),
		Email:     req.Email,
		Name:      getNameFromEmail(req.Email),
		CreatedAt: time.Now(),
	}
	h.sessionMu.Unlock()

	// Return token + user info
	user := h.getDevUser(req.Email)
	user.ID = h.sessions[token].UserID
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user":  user,
	})
}

// handleRegister handles POST /auth/register.
// In dev mode, accepts any input and returns a Bearer token.
func (h *DevAuthHandlers) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		h.handleCORS(w)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email == "" {
		h.writeError(w, http.StatusBadRequest, "Email is required")
		return
	}

	// In dev mode, accept any registration
	h.logger.Info("dev auth: register", "email", req.Email, "name", req.Name)

	name := req.Name
	if name == "" {
		name = getNameFromEmail(req.Email)
	}

	// Create a session keyed by token
	token := "tok_" + randomString(32)
	h.sessionMu.Lock()
	h.sessions[token] = &DevSession{
		UserID:    "dev-user-" + randomString(8),
		Email:     req.Email,
		Name:      name,
		CreatedAt: time.Now(),
	}
	h.sessionMu.Unlock()

	// Return token + user info
	user := h.getDevUser(req.Email)
	user.ID = h.sessions[token].UserID
	user.Name = name
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user":  user,
	})
}

// handleMe handles GET /auth/me.
// Returns the current user if a valid Bearer token is provided.
func (h *DevAuthHandlers) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		h.handleCORS(w)
		return
	}

	token := extractAuthToken(r)
	if token == "" {
		h.writeError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	h.sessionMu.RLock()
	session, exists := h.sessions[token]
	h.sessionMu.RUnlock()

	if !exists {
		h.writeError(w, http.StatusUnauthorized, "Invalid or expired token")
		return
	}

	user := h.getDevUser(session.Email)
	user.ID = session.UserID
	user.Name = session.Name
	h.writeJSON(w, http.StatusOK, user)
}

// handleLogout handles POST /auth/logout.
// Invalidates the Bearer token.
func (h *DevAuthHandlers) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		h.handleCORS(w)
		return
	}

	token := extractAuthToken(r)
	if token != "" {
		h.sessionMu.Lock()
		delete(h.sessions, token)
		h.sessionMu.Unlock()
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

// handleForgotPassword handles POST /auth/forgot.
// In dev mode, just returns success.
func (h *DevAuthHandlers) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		h.handleCORS(w)
		return
	}

	h.logger.Info("dev auth: forgot password (no-op in dev mode)")
	h.writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": "If your email exists, you will receive a reset link (dev mode: no email sent)",
	})
}

// handleResetPassword handles POST /auth/reset.
// In dev mode, just returns success.
func (h *DevAuthHandlers) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		h.handleCORS(w)
		return
	}

	h.logger.Info("dev auth: reset password (no-op in dev mode)")
	h.writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": "Password reset successful (dev mode: password not actually changed)",
	})
}

// getDevUser returns a dev user with generous plan limits.
func (h *DevAuthHandlers) getDevUser(email string) DevUser {
	return DevUser{
		ID:     "dev-user-" + randomString(8),
		Email:  email,
		Name:   getNameFromEmail(email),
		PlanID: "dev-unlimited",
		PlanLimits: DevPlanLimits{
			MaxDeployments: 100,
			MaxCPUCores:    16,
			MaxMemoryMB:    32768,
			MaxDiskGB:      500,
		},
	}
}

// handleCORS handles CORS preflight requests.
func (h *DevAuthHandlers) handleCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Auth-Token")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.WriteHeader(http.StatusOK)
}

// writeJSON writes a JSON response.
func (h *DevAuthHandlers) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes a JSON error response.
func (h *DevAuthHandlers) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"message": message,
		},
	})
}

// extractAuthToken extracts the auth token from the X-Auth-Token header.
// Uses X-Auth-Token because APIGate strips the standard Authorization header.
func extractAuthToken(r *http.Request) string {
	return r.Header.Get("X-Auth-Token")
}

// getNameFromEmail extracts a display name from an email address.
func getNameFromEmail(email string) string {
	// Take the part before @
	for i, c := range email {
		if c == '@' {
			return email[:i]
		}
	}
	return email
}
