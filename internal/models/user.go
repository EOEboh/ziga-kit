package models

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// ErrNotFound is returned by query methods when no row is found.
// Handlers should map this to a 404, not a 500.
var ErrNotFound = errors.New("record not found")

// ErrDuplicateEmail is returned by CreateUser when the email already exists.
var ErrDuplicateEmail = errors.New("email already in use")

// SubscriptionTier mirrors the Postgres enum.
type SubscriptionTier string

const (
	TierFree SubscriptionTier = "free"
	TierPro  SubscriptionTier = "pro"
)

// User maps to the `users` table.
type User struct {
	ID           string           `json:"id"`
	Email        string           `json:"email"`
	PasswordHash string           `json:"-"` // never serialise
	FullName     *string          `json:"full_name,omitempty"`
	Tier         SubscriptionTier `json:"tier"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

// ─── Queries ─────────────────────────────────────────────────────────────────

// CreateUser inserts a new user row and returns the created record.
// Returns ErrDuplicateEmail if the email is already taken.
func CreateUser(ctx context.Context, db *pgxpool.Pool, email, password, fullName string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	var u User
	err = db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, full_name)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, full_name, tier, created_at, updated_at
	`, email, string(hash), nullableString(fullName)).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FullName,
		&u.Tier, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if isDuplicateKeyError(err) {
			return nil, ErrDuplicateEmail
		}
		return nil, err
	}

	return &u, nil
}

// GetUserByEmail fetches a user by email.
// Returns ErrNotFound if no matching row exists.
func GetUserByEmail(ctx context.Context, db *pgxpool.Pool, email string) (*User, error) {
	var u User
	err := db.QueryRow(ctx, `
		SELECT id, email, password_hash, full_name, tier, created_at, updated_at
		FROM users
		WHERE email = $1
	`, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FullName,
		&u.Tier, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &u, nil
}

// GetUserByID fetches a user by primary key.
func GetUserByID(ctx context.Context, db *pgxpool.Pool, id string) (*User, error) {
	var u User
	err := db.QueryRow(ctx, `
		SELECT id, email, password_hash, full_name, tier, created_at, updated_at
		FROM users
		WHERE id = $1
	`, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FullName,
		&u.Tier, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &u, nil
}

// CheckPassword returns nil if the plaintext password matches the stored hash.
func (u *User) CheckPassword(plaintext string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(plaintext))
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// nullableString returns nil for an empty string so Postgres stores NULL.
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// isDuplicateKeyError detects Postgres unique-violation (code 23505).
func isDuplicateKeyError(err error) bool {
	return err != nil && (contains(err.Error(), "23505") || contains(err.Error(), "duplicate key"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}