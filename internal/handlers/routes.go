package handlers

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/EOEboh/ziga-kit/internal/config"
	"github.com/EOEboh/ziga-kit/internal/db"
	appMiddleware "github.com/EOEboh/ziga-kit/internal/middleware"
)

// NewRouter builds and returns the fully-configured Chi router.
// All middleware, route groups, and handler bindings live here.
func NewRouter(db *db.Pool, cfg *config.Config) http.Handler {
	r := chi.NewRouter()

	// ── Global middleware stack ────────────────────────────────────────────────
	// Order matters: outermost middleware runs first on the way in,
	// last on the way out.

	// Recover from panics and return a 500 instead of crashing the process.
	r.Use(chiMiddleware.Recoverer)

	// Structured request logging (our custom slog-based logger).
	r.Use(appMiddleware.Logger)

	// CORS — allow the Next.js frontend origin.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.FrontendURL},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // preflight cache in seconds
	}))

	// Request ID — attaches a unique X-Request-ID to every request.
	// Useful for correlating logs with error reports.
	r.Use(chiMiddleware.RequestID)

	// ── Initialise handlers ───────────────────────────────────────────────────
	authHandler := NewAuthHandler(db, cfg)
	projectHandler := NewProjectHandler(db, cfg)

	// ── Routes ────────────────────────────────────────────────────────────────

	// Health check — no version prefix, used by Railway/Render uptime checks.
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","env":%q}`, cfg.AppEnv)
	})

	r.Route("/api/v1", func(r chi.Router) {

		// ── Public routes (no auth required) ──────────────────────────────────

		// Auth
		r.Post("/auth/signup", authHandler.Signup)
		r.Post("/auth/login", authHandler.Login)

		// Client-facing project view (shareable link)
		r.Get("/public/projects/{token}", projectHandler.GetPublic)

		// Client feedback submission (also public)
		r.Post("/public/deliverables/{deliverableID}/feedback", SubmitFeedback(db, cfg))

		// ── Protected routes (JWT required) ───────────────────────────────────
		r.Group(func(r chi.Router) {
			r.Use(appMiddleware.Authenticate(cfg.JWTSecret))

			// Auth
			r.Get("/auth/me", authHandler.Me)

			// Projects
			r.Get("/projects", projectHandler.List)
			r.Post("/projects", projectHandler.Create)
			r.Get("/projects/{id}", projectHandler.Get)

			// Deliverables (scoped under a project)
			r.Post("/projects/{projectID}/deliverables", CreateDeliverable(db, cfg))
			r.Get("/projects/{projectID}/deliverables", ListDeliverables(db, cfg))
			r.Patch("/projects/{projectID}/deliverables/{id}/status", UpdateDeliverableStatus(db, cfg))

			// Milestones
			r.Post("/projects/{projectID}/milestones", CreateMilestone(db, cfg))
			r.Get("/projects/{projectID}/milestones", ListMilestones(db, cfg))
			r.Patch("/projects/{projectID}/milestones/{id}", ToggleMilestone(db, cfg))
		})
	})

	return r
}
