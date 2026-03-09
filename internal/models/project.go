package models

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Project maps to the `projects` table.
type Project struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	Title          string     `json:"title"`
	Description    *string    `json:"description,omitempty"`
	Deadline       *string    `json:"deadline,omitempty"` // ISO date string YYYY-MM-DD
	PublicToken    string     `json:"public_token"`
	MilestoneIndex int        `json:"milestone_index"`
	BrandColor     *string    `json:"brand_color,omitempty"`
	BrandLogoKey   *string    `json:"-"` // resolved to URL in handler
	ArchivedAt     *time.Time `json:"archived_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// CreateProjectInput holds the caller-supplied fields for a new project.
type CreateProjectInput struct {
	UserID      string
	Title       string
	Description string
	Deadline    string // YYYY-MM-DD, may be empty
}

// CreateProject inserts a new project row and returns it.
// The public_token is generated here and guaranteed unique via DB constraint.
func CreateProject(ctx context.Context, db *pgxpool.Pool, in CreateProjectInput) (*Project, error) {
	token, err := generateToken(16) // 32 hex chars
	if err != nil {
		return nil, err
	}

	var p Project
	err = db.QueryRow(ctx, `
		INSERT INTO projects (user_id, title, description, deadline, public_token)
		VALUES ($1, $2, $3, NULLIF($4, '')::date, $5)
		RETURNING id, user_id, title, description,
		          deadline::text, public_token, milestone_index,
		          brand_color, brand_logo_key, archived_at, created_at, updated_at
	`, in.UserID, in.Title, nullableString(in.Description), in.Deadline, token,
	).Scan(
		&p.ID, &p.UserID, &p.Title, &p.Description,
		&p.Deadline, &p.PublicToken, &p.MilestoneIndex,
		&p.BrandColor, &p.BrandLogoKey, &p.ArchivedAt,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

// ListProjectsByUser returns all non-archived projects for a user, newest first.
func ListProjectsByUser(ctx context.Context, db *pgxpool.Pool, userID string) ([]*Project, error) {
	rows, err := db.Query(ctx, `
		SELECT id, user_id, title, description,
		       deadline::text, public_token, milestone_index,
		       brand_color, brand_logo_key, archived_at, created_at, updated_at
		FROM projects
		WHERE user_id = $1
		  AND archived_at IS NULL
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(
			&p.ID, &p.UserID, &p.Title, &p.Description,
			&p.Deadline, &p.PublicToken, &p.MilestoneIndex,
			&p.BrandColor, &p.BrandLogoKey, &p.ArchivedAt,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		projects = append(projects, &p)
	}

	return projects, rows.Err()
}

// GetProjectByToken fetches a project by its public token (used on the public
// client-facing page — no auth required).
func GetProjectByToken(ctx context.Context, db *pgxpool.Pool, token string) (*Project, error) {
	var p Project
	err := db.QueryRow(ctx, `
		SELECT id, user_id, title, description,
		       deadline::text, public_token, milestone_index,
		       brand_color, brand_logo_key, archived_at, created_at, updated_at
		FROM projects
		WHERE public_token = $1
		  AND archived_at IS NULL
	`, token).Scan(
		&p.ID, &p.UserID, &p.Title, &p.Description,
		&p.Deadline, &p.PublicToken, &p.MilestoneIndex,
		&p.BrandColor, &p.BrandLogoKey, &p.ArchivedAt,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &p, nil
}

// GetProjectByID fetches a project owned by the given user.
// Returns ErrNotFound if the project doesn't exist OR belongs to a different user.
func GetProjectByID(ctx context.Context, db *pgxpool.Pool, id, userID string) (*Project, error) {
	var p Project
	err := db.QueryRow(ctx, `
		SELECT id, user_id, title, description,
		       deadline::text, public_token, milestone_index,
		       brand_color, brand_logo_key, archived_at, created_at, updated_at
		FROM projects
		WHERE id = $1
		  AND user_id = $2
		  AND archived_at IS NULL
	`, id, userID).Scan(
		&p.ID, &p.UserID, &p.Title, &p.Description,
		&p.Deadline, &p.PublicToken, &p.MilestoneIndex,
		&p.BrandColor, &p.BrandLogoKey, &p.ArchivedAt,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &p, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// generateToken returns a URL-safe random hex string of `byteLen` bytes.
func generateToken(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
