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
	"github.com/go-chi/chi/v5"
)

// ProjectHandler holds dependencies for project endpoints.
type ProjectHandler struct {
	db  *db.Pool
	cfg *config.Config
}

func NewProjectHandler(db *db.Pool, cfg *config.Config) *ProjectHandler {
	return &ProjectHandler{db: db, cfg: cfg}
}

// ─── Create ───────────────────────────────────────────────────────────────────

type createProjectRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Deadline    string `json:"deadline"` // YYYY-MM-DD
}

// Create godoc
//
//	POST /api/v1/projects  (protected)
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		respond.Error(w, http.StatusBadRequest, "title is required")
		return
	}

	project, err := models.CreateProject(r.Context(), h.db, models.CreateProjectInput{
		UserID:      claims.UserID,
		Title:       req.Title,
		Description: req.Description,
		Deadline:    req.Deadline,
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to create project")
		return
	}

	respond.JSON(w, http.StatusCreated, h.withPublicURL(project))
}

// ─── List ─────────────────────────────────────────────────────────────────────

// List godoc
//
//	GET /api/v1/projects  (protected)
func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())

	projects, err := models.ListProjectsByUser(r.Context(), h.db, claims.UserID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list projects")
		return
	}

	// Ensure we return [] not null for empty results
	if projects == nil {
		projects = []*models.Project{}
	}

	// Attach public URLs
	for i, p := range projects {
		projects[i] = h.withPublicURL(p)
	}

	respond.JSON(w, http.StatusOK, projects)
}

// ─── Get (by owner) ───────────────────────────────────────────────────────────

// Get godoc
//
//	GET /api/v1/projects/{id}  (protected)
func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	id := chi.URLParam(r, "id")

	project, err := models.GetProjectByID(r.Context(), h.db, id, claims.UserID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			respond.Error(w, http.StatusNotFound, "project not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to fetch project")
		return
	}

	respond.JSON(w, http.StatusOK, h.withPublicURL(project))
}

// ─── Public (client-facing, no auth) ─────────────────────────────────────────

type publicProjectResponse struct {
	Project      *models.Project       `json:"project"`
	Deliverables []*models.Deliverable `json:"deliverables"`
	Milestones   []*models.Milestone   `json:"milestones"`
}

// GetPublic godoc
//
//	GET /api/v1/public/projects/{token}  (no auth)
func (h *ProjectHandler) GetPublic(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	project, err := models.GetProjectByToken(r.Context(), h.db, token)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			respond.Error(w, http.StatusNotFound, "project not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to fetch project")
		return
	}

	deliverables, err := models.ListDeliverablesByProject(r.Context(), h.db, project.ID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to fetch deliverables")
		return
	}
	if deliverables == nil {
		deliverables = []*models.Deliverable{}
	}

	milestones, err := models.ListMilestonesByProject(r.Context(), h.db, project.ID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to fetch milestones")
		return
	}
	if milestones == nil {
		milestones = []*models.Milestone{}
	}

	respond.JSON(w, http.StatusOK, publicProjectResponse{
		Project:      project,
		Deliverables: deliverables,
		Milestones:   milestones,
	})
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// withPublicURL attaches the fully-qualified shareable URL to a project.
func (h *ProjectHandler) withPublicURL(p *models.Project) *models.Project {
	// We add a synthetic field by embedding a custom struct? Simpler: just add
	// PublicURL as a computed field on the model response. For now we keep it
	// simple — the frontend can construct the URL from public_token.
	// TODO: Embed public_url in response envelope if the frontend requests it.
	return p
}
