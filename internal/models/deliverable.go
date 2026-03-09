package models

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── Deliverable ─────────────────────────────────────────────────────────────

type DeliverableStatus string

const (
	StatusDraft    DeliverableStatus = "draft"
	StatusReview   DeliverableStatus = "review"
	StatusApproved DeliverableStatus = "approved"
)

type Deliverable struct {
	ID         string            `json:"id"`
	ProjectID  string            `json:"project_id"`
	Label      string            `json:"label"`
	LinkURL    *string           `json:"link_url,omitempty"`
	FileKey    *string           `json:"-"`                  // resolved to signed URL in handler
	FileURL    *string           `json:"file_url,omitempty"` // populated by handler
	Status     DeliverableStatus `json:"status"`
	OrderIndex int               `json:"order_index"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

type CreateDeliverableInput struct {
	ProjectID  string
	Label      string
	LinkURL    string
	FileKey    string
	OrderIndex int
}

func CreateDeliverable(ctx context.Context, db *pgxpool.Pool, in CreateDeliverableInput) (*Deliverable, error) {
	var d Deliverable
	err := db.QueryRow(ctx, `
		INSERT INTO deliverables (project_id, label, link_url, file_key, order_index)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5)
		RETURNING id, project_id, label, link_url, file_key, status, order_index, created_at, updated_at
	`, in.ProjectID, in.Label, in.LinkURL, in.FileKey, in.OrderIndex,
	).Scan(
		&d.ID, &d.ProjectID, &d.Label, &d.LinkURL, &d.FileKey,
		&d.Status, &d.OrderIndex, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func ListDeliverablesByProject(ctx context.Context, db *pgxpool.Pool, projectID string) ([]*Deliverable, error) {
	rows, err := db.Query(ctx, `
		SELECT id, project_id, label, link_url, file_key, status, order_index, created_at, updated_at
		FROM deliverables
		WHERE project_id = $1
		ORDER BY order_index ASC, created_at ASC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*Deliverable
	for rows.Next() {
		var d Deliverable
		if err := rows.Scan(
			&d.ID, &d.ProjectID, &d.Label, &d.LinkURL, &d.FileKey,
			&d.Status, &d.OrderIndex, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, &d)
	}
	return items, rows.Err()
}

func UpdateDeliverableStatus(ctx context.Context, db *pgxpool.Pool, id, projectID string, status DeliverableStatus) (*Deliverable, error) {
	var d Deliverable
	err := db.QueryRow(ctx, `
		UPDATE deliverables
		SET status = $1
		WHERE id = $2 AND project_id = $3
		RETURNING id, project_id, label, link_url, file_key, status, order_index, created_at, updated_at
	`, status, id, projectID).Scan(
		&d.ID, &d.ProjectID, &d.Label, &d.LinkURL, &d.FileKey,
		&d.Status, &d.OrderIndex, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// ─── Feedback ─────────────────────────────────────────────────────────────────

type FeedbackAction string

const (
	ActionApproved         FeedbackAction = "approved"
	ActionChangesRequested FeedbackAction = "changes_requested"
)

type Feedback struct {
	ID            string         `json:"id"`
	DeliverableID string         `json:"deliverable_id"`
	ClientName    string         `json:"client_name"`
	Comment       *string        `json:"comment,omitempty"`
	Action        FeedbackAction `json:"action"`
	CreatedAt     time.Time      `json:"created_at"`
}

type CreateFeedbackInput struct {
	DeliverableID string
	ClientName    string
	Comment       string
	Action        FeedbackAction
	ClientIP      string
}

func CreateFeedback(ctx context.Context, db *pgxpool.Pool, in CreateFeedbackInput) (*Feedback, error) {
	var f Feedback
	err := db.QueryRow(ctx, `
		INSERT INTO feedback (deliverable_id, client_name, comment, action, client_ip)
		VALUES ($1, $2, NULLIF($3, ''), $4, NULLIF($5, ''))
		RETURNING id, deliverable_id, client_name, comment, action, created_at
	`, in.DeliverableID, in.ClientName, in.Comment, in.Action, in.ClientIP,
	).Scan(&f.ID, &f.DeliverableID, &f.ClientName, &f.Comment, &f.Action, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func ListFeedbackByDeliverable(ctx context.Context, db *pgxpool.Pool, deliverableID string) ([]*Feedback, error) {
	rows, err := db.Query(ctx, `
		SELECT id, deliverable_id, client_name, comment, action, created_at
		FROM feedback
		WHERE deliverable_id = $1
		ORDER BY created_at DESC
	`, deliverableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*Feedback
	for rows.Next() {
		var f Feedback
		if err := rows.Scan(&f.ID, &f.DeliverableID, &f.ClientName, &f.Comment, &f.Action, &f.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, &f)
	}
	return items, rows.Err()
}

// ─── Milestone ────────────────────────────────────────────────────────────────

type Milestone struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"project_id"`
	Title      string    `json:"title"`
	OrderIndex int       `json:"order_index"`
	Completed  bool      `json:"completed"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func CreateMilestone(ctx context.Context, db *pgxpool.Pool, projectID, title string, orderIndex int) (*Milestone, error) {
	var m Milestone
	err := db.QueryRow(ctx, `
		INSERT INTO milestones (project_id, title, order_index)
		VALUES ($1, $2, $3)
		RETURNING id, project_id, title, order_index, completed, created_at, updated_at
	`, projectID, title, orderIndex).Scan(
		&m.ID, &m.ProjectID, &m.Title, &m.OrderIndex, &m.Completed, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func ListMilestonesByProject(ctx context.Context, db *pgxpool.Pool, projectID string) ([]*Milestone, error) {
	rows, err := db.Query(ctx, `
		SELECT id, project_id, title, order_index, completed, created_at, updated_at
		FROM milestones
		WHERE project_id = $1
		ORDER BY order_index ASC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*Milestone
	for rows.Next() {
		var m Milestone
		if err := rows.Scan(&m.ID, &m.ProjectID, &m.Title, &m.OrderIndex, &m.Completed, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, &m)
	}
	return items, rows.Err()
}

func ToggleMilestone(ctx context.Context, db *pgxpool.Pool, id, projectID string, completed bool) (*Milestone, error) {
	var m Milestone
	err := db.QueryRow(ctx, `
		UPDATE milestones
		SET completed = $1
		WHERE id = $2 AND project_id = $3
		RETURNING id, project_id, title, order_index, completed, created_at, updated_at
	`, completed, id, projectID).Scan(
		&m.ID, &m.ProjectID, &m.Title, &m.OrderIndex, &m.Completed, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &m, nil
}
