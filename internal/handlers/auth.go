package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/EOEboh/ziga-kit/internal/config"
	"github.com/EOEboh/ziga-kit/internal/db"
	"github.com/EOEboh/ziga-kit/internal/handlers/respond"
	"github.com/EOEboh/ziga-kit/internal/middleware"
	"github.com/EOEboh/ziga-kit/internal/models"
)

// AuthHandler holds the dependencies for auth endpoints.
type AuthHandler struct {
	db  *db.Pool
	cfg *config.Config
}

// NewAuthHandler constructs an AuthHandler with its dependencies.
func NewAuthHandler(db *db.Pool, cfg *config.Config) *AuthHandler {
	return &AuthHandler{db: db, cfg: cfg}
}

// ─── Signup ───────────────────────────────────────────────────────────────────

type signupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

type authResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

// Signup godoc
//
//	POST /api/v1/auth/signup
//	Body: { "email": "...", "password": "...", "full_name": "..." }
//	Returns: { "token": "...", "user": { ... } }
func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// ── Validation ────────────────────────────────────────────────────────────
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.FullName = strings.TrimSpace(req.FullName)

	if req.Email == "" || !strings.Contains(req.Email, "@") {
		respond.Error(w, http.StatusBadRequest, "a valid email is required")
		return
	}
	if len(req.Password) < 8 {
		respond.Error(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	// ── Persist ───────────────────────────────────────────────────────────────
	user, err := models.CreateUser(r.Context(), h.db, req.Email, req.Password, req.FullName)
	if err != nil {
		if errors.Is(err, models.ErrDuplicateEmail) {
			respond.Error(w, http.StatusConflict, "an account with this email already exists")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to create account")
		return
	}

	// ── Issue token ───────────────────────────────────────────────────────────
	token, err := middleware.GenerateToken(
		user.ID, user.Email, string(user.Tier),
		h.cfg.JWTSecret, h.cfg.JWTExpiryHours,
	)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	respond.JSON(w, http.StatusCreated, authResponse{Token: token, User: user})
}

// ─── Login ────────────────────────────────────────────────────────────────────

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login godoc
//
//	POST /api/v1/auth/login
//	Body: { "email": "...", "password": "..." }
//	Returns: { "token": "...", "user": { ... } }
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if req.Email == "" || req.Password == "" {
		respond.Error(w, http.StatusBadRequest, "email and password are required")
		return
	}

	// ── Lookup ────────────────────────────────────────────────────────────────
	user, err := models.GetUserByEmail(r.Context(), h.db, req.Email)
	if err != nil {
		// Return the same message whether the user doesn't exist OR the password
		// is wrong — prevents email enumeration attacks.
		respond.Error(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if err := user.CheckPassword(req.Password); err != nil {
		respond.Error(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	// ── Issue token ───────────────────────────────────────────────────────────
	token, err := middleware.GenerateToken(
		user.ID, user.Email, string(user.Tier),
		h.cfg.JWTSecret, h.cfg.JWTExpiryHours,
	)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	respond.JSON(w, http.StatusOK, authResponse{Token: token, User: user})
}

// ─── Me ───────────────────────────────────────────────────────────────────────

// Me godoc
//
//	GET /api/v1/auth/me   (protected)
//	Returns the authenticated user's profile.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	user, err := models.GetUserByID(r.Context(), h.db, claims.UserID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			respond.Error(w, http.StatusNotFound, "user not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to fetch user")
		return
	}

	respond.JSON(w, http.StatusOK, user)
}
