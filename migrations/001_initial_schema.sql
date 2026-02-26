-- ============================================================
-- Migration: 001_initial_schema
-- Created:   2024-01-01
-- Description: Bootstrap schema for Ziga-Kit MVP
-- ============================================================
-- Run with: make migrate
-- or manually: psql $DATABASE_URL -f migrations/001_initial_schema.sql
-- ============================================================

-- ── Extensions ───────────────────────────────────────────────────────────────

-- pgcrypto gives us gen_random_uuid() for UUID primary keys.
-- UUIDs are preferred over serial integers for public-facing IDs —
-- they're non-enumerable and safe to expose in URLs / API responses.
CREATE EXTENSION IF NOT EXISTS "pgcrypto";


-- ── ENUM types ───────────────────────────────────────────────────────────────

-- Deliverable workflow state
CREATE TYPE deliverable_status AS ENUM (
    'draft',
    'review',       -- "Ready for Review"
    'approved'
);

-- Client feedback action
CREATE TYPE feedback_action AS ENUM (
    'approved',
    'changes_requested'
);

-- Subscription tier (used for feature gating)
CREATE TYPE subscription_tier AS ENUM (
    'free',
    'pro'
);


-- ── users ────────────────────────────────────────────────────────────────────

CREATE TABLE users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,

    -- Display / branding
    full_name     TEXT,

    -- Subscription state
    tier          subscription_tier NOT NULL DEFAULT 'free',

    -- Stripe / Paystack customer IDs — nullable until the user subscribes
    stripe_customer_id    TEXT UNIQUE,
    paystack_customer_id  TEXT UNIQUE,

    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Fast lookup by email (login path)
CREATE INDEX idx_users_email ON users (email);


-- ── projects ─────────────────────────────────────────────────────────────────

CREATE TABLE projects (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    title           TEXT        NOT NULL,
    description     TEXT,
    deadline        DATE,

    -- The token that powers the public shareable URL
    -- e.g. /p/<public_token>
    -- Generated server-side with crypto/rand, stored here for fast lookup.
    public_token    TEXT        NOT NULL UNIQUE,

    -- Index of the currently active milestone (0-based).
    -- Freelancer advances this manually.
    milestone_index INT         NOT NULL DEFAULT 0,

    -- Optional branding overrides (Pro tier)
    brand_color     TEXT,           -- hex, e.g. "#7C3AED"
    brand_logo_key  TEXT,           -- R2 file key

    -- Soft-delete: archived projects stop appearing but data is retained
    archived_at     TIMESTAMPTZ,

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_projects_user_id     ON projects (user_id);
CREATE INDEX idx_projects_public_token ON projects (public_token);


-- ── deliverables ─────────────────────────────────────────────────────────────

CREATE TABLE deliverables (
    id          UUID                PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID                NOT NULL REFERENCES projects(id) ON DELETE CASCADE,

    label       TEXT                NOT NULL,

    -- Exactly one of link_url or file_key will be set.
    -- A CHECK constraint enforces this.
    link_url    TEXT,
    file_key    TEXT,               -- Cloudflare R2 object key

    status      deliverable_status  NOT NULL DEFAULT 'draft',

    -- Display order within a project (freelancer can reorder)
    order_index INT                 NOT NULL DEFAULT 0,

    created_at  TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ         NOT NULL DEFAULT NOW(),

    CONSTRAINT deliverable_has_source CHECK (
        (link_url IS NOT NULL AND file_key IS NULL) OR
        (link_url IS NULL AND file_key IS NOT NULL)
    )
);

CREATE INDEX idx_deliverables_project_id ON deliverables (project_id);


-- ── feedback ─────────────────────────────────────────────────────────────────

-- Submitted by clients (unauthenticated). One row per submission.
-- We don't enforce one-per-deliverable at the DB level because clients
-- may re-submit after a revision. The application layer surfaces the latest.
CREATE TABLE feedback (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    deliverable_id  UUID            NOT NULL REFERENCES deliverables(id) ON DELETE CASCADE,

    client_name     TEXT            NOT NULL,
    comment         TEXT,
    action          feedback_action NOT NULL,

    -- Capture IP for basic abuse prevention (hash it before storing if you
    -- want to be GDPR-friendly)
    client_ip       TEXT,

    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_feedback_deliverable_id ON feedback (deliverable_id);


-- ── milestones ───────────────────────────────────────────────────────────────

CREATE TABLE milestones (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,

    title       TEXT        NOT NULL,
    order_index INT         NOT NULL DEFAULT 0,
    completed   BOOLEAN     NOT NULL DEFAULT FALSE,

    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_milestones_project_id ON milestones (project_id);

-- Unique ordering per project — no two milestones can share the same position
CREATE UNIQUE INDEX idx_milestones_order ON milestones (project_id, order_index);


-- ── updated_at trigger ───────────────────────────────────────────────────────
-- Automatically keeps updated_at current on any UPDATE.

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_projects_updated_at
    BEFORE UPDATE ON projects
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_deliverables_updated_at
    BEFORE UPDATE ON deliverables
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_milestones_updated_at
    BEFORE UPDATE ON milestones
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();