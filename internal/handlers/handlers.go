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

// ─── Deliverables ─────────────────────────────────────────────────────────────

// CreateDeliverable godoc
//
//	POST /api/v1/projects/{projectID}/deliverables  (protected)
func CreateDeliverable(db *db.Pool, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := middleware.ClaimsFromContext(r.Context())
		projectID := chi.URLParam(r, "projectID")

		// Confirm the project belongs to the requesting user
		_, err := models.GetProjectByID(r.Context(), db, projectID, claims.UserID)
		if err != nil {
			if errors.Is(err, models.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "project not found")
				return
			}
			respond.Error(w, http.StatusInternalServerError, "server error")
			return
		}

		var req struct {
			Label      string `json:"label"`
			LinkURL    string `json:"link_url"`
			FileKey    string `json:"file_key"`
			OrderIndex int    `json:"order_index"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respond.Error(w, http.StatusBadRequest, "invalid request body")
			return
		}

		req.Label = strings.TrimSpace(req.Label)
		if req.Label == "" {
			respond.Error(w, http.StatusBadRequest, "label is required")
			return
		}
		if req.LinkURL == "" && req.FileKey == "" {
			respond.Error(w, http.StatusBadRequest, "either link_url or file_key is required")
			return
		}
		if req.LinkURL != "" && req.FileKey != "" {
			respond.Error(w, http.StatusBadRequest, "provide only one of link_url or file_key, not both")
			return
		}

		d, err := models.CreateDeliverable(r.Context(), db, models.CreateDeliverableInput{
			ProjectID:  projectID,
			Label:      req.Label,
			LinkURL:    req.LinkURL,
			FileKey:    req.FileKey,
			OrderIndex: req.OrderIndex,
		})
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to create deliverable")
			return
		}

		respond.JSON(w, http.StatusCreated, d)
	}
}

// ListDeliverables godoc
//
//	GET /api/v1/projects/{projectID}/deliverables  (protected)
func ListDeliverables(db *db.Pool, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := middleware.ClaimsFromContext(r.Context())
		projectID := chi.URLParam(r, "projectID")

		_, err := models.GetProjectByID(r.Context(), db, projectID, claims.UserID)
		if err != nil {
			if errors.Is(err, models.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "project not found")
				return
			}
			respond.Error(w, http.StatusInternalServerError, "server error")
			return
		}

		deliverables, err := models.ListDeliverablesByProject(r.Context(), db, projectID)
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to list deliverables")
			return
		}
		if deliverables == nil {
			deliverables = []*models.Deliverable{}
		}

		respond.JSON(w, http.StatusOK, deliverables)
	}
}

// UpdateDeliverableStatus godoc
//
//	PATCH /api/v1/projects/{projectID}/deliverables/{id}/status  (protected)
func UpdateDeliverableStatus(db *db.Pool, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := middleware.ClaimsFromContext(r.Context())
		projectID := chi.URLParam(r, "projectID")
		id := chi.URLParam(r, "id")

		_, err := models.GetProjectByID(r.Context(), db, projectID, claims.UserID)
		if err != nil {
			if errors.Is(err, models.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "project not found")
				return
			}
			respond.Error(w, http.StatusInternalServerError, "server error")
			return
		}

		var req struct {
			Status models.DeliverableStatus `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respond.Error(w, http.StatusBadRequest, "invalid request body")
			return
		}

		switch req.Status {
		case models.StatusDraft, models.StatusReview, models.StatusApproved:
			// valid
		default:
			respond.Error(w, http.StatusBadRequest, "status must be one of: draft, review, approved")
			return
		}

		d, err := models.UpdateDeliverableStatus(r.Context(), db, id, projectID, req.Status)
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to update status")
			return
		}

		respond.JSON(w, http.StatusOK, d)
	}
}

// ─── Feedback (public) ────────────────────────────────────────────────────────

// SubmitFeedback godoc
//
//	POST /api/v1/public/deliverables/{deliverableID}/feedback  (no auth)
func SubmitFeedback(db *db.Pool, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deliverableID := chi.URLParam(r, "deliverableID")

		var req struct {
			ClientName string                `json:"client_name"`
			Comment    string                `json:"comment"`
			Action     models.FeedbackAction `json:"action"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respond.Error(w, http.StatusBadRequest, "invalid request body")
			return
		}

		req.ClientName = strings.TrimSpace(req.ClientName)
		if req.ClientName == "" {
			respond.Error(w, http.StatusBadRequest, "client_name is required")
			return
		}
		if req.Action != models.ActionApproved && req.Action != models.ActionChangesRequested {
			respond.Error(w, http.StatusBadRequest, "action must be 'approved' or 'changes_requested'")
			return
		}

		feedback, err := models.CreateFeedback(r.Context(), db, models.CreateFeedbackInput{
			DeliverableID: deliverableID,
			ClientName:    req.ClientName,
			Comment:       req.Comment,
			Action:        req.Action,
			ClientIP:      realClientIP(r),
		})
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to submit feedback")
			return
		}

		// TODO: trigger email notification to the freelancer here (step 6)

		respond.JSON(w, http.StatusCreated, feedback)
	}
}

// ─── Milestones ───────────────────────────────────────────────────────────────

// CreateMilestone godoc
//
//	POST /api/v1/projects/{projectID}/milestones  (protected)
func CreateMilestone(db *db.Pool, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := middleware.ClaimsFromContext(r.Context())
		projectID := chi.URLParam(r, "projectID")

		_, err := models.GetProjectByID(r.Context(), db, projectID, claims.UserID)
		if err != nil {
			if errors.Is(err, models.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "project not found")
				return
			}
			respond.Error(w, http.StatusInternalServerError, "server error")
			return
		}

		var req struct {
			Title      string `json:"title"`
			OrderIndex int    `json:"order_index"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respond.Error(w, http.StatusBadRequest, "invalid request body")
			return
		}

		req.Title = strings.TrimSpace(req.Title)
		if req.Title == "" {
			respond.Error(w, http.StatusBadRequest, "title is required")
			return
		}

		m, err := models.CreateMilestone(r.Context(), db, projectID, req.Title, req.OrderIndex)
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to create milestone")
			return
		}

		respond.JSON(w, http.StatusCreated, m)
	}
}

// ListMilestones godoc
//
//	GET /api/v1/projects/{projectID}/milestones  (protected)
func ListMilestones(db *db.Pool, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := middleware.ClaimsFromContext(r.Context())
		projectID := chi.URLParam(r, "projectID")

		_, err := models.GetProjectByID(r.Context(), db, projectID, claims.UserID)
		if err != nil {
			if errors.Is(err, models.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "project not found")
				return
			}
			respond.Error(w, http.StatusInternalServerError, "server error")
			return
		}

		milestones, err := models.ListMilestonesByProject(r.Context(), db, projectID)
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to list milestones")
			return
		}
		if milestones == nil {
			milestones = []*models.Milestone{}
		}

		respond.JSON(w, http.StatusOK, milestones)
	}
}

// ToggleMilestone godoc
//
//	PATCH /api/v1/projects/{projectID}/milestones/{id}  (protected)
func ToggleMilestone(db *db.Pool, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := middleware.ClaimsFromContext(r.Context())
		projectID := chi.URLParam(r, "projectID")
		id := chi.URLParam(r, "id")

		_, err := models.GetProjectByID(r.Context(), db, projectID, claims.UserID)
		if err != nil {
			if errors.Is(err, models.ErrNotFound) {
				respond.Error(w, http.StatusNotFound, "project not found")
				return
			}
			respond.Error(w, http.StatusInternalServerError, "server error")
			return
		}

		var req struct {
			Completed bool `json:"completed"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respond.Error(w, http.StatusBadRequest, "invalid request body")
			return
		}

		m, err := models.ToggleMilestone(r.Context(), db, id, projectID, req.Completed)
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to update milestone")
			return
		}

		respond.JSON(w, http.StatusOK, m)
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func realClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		for i, ch := range ip {
			if ch == ',' {
				return ip[:i]
			}
		}
		return ip
	}
	return r.RemoteAddr
}
