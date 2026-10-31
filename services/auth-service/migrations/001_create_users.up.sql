-- Auth DB: users table.
-- See docs/architecture/DATABASE_DESIGN.md §2.
--
-- Role is modeled as a simple enum column on the user row rather than a
-- separate Role/assignment table, per the MVP simplification documented
-- in DATABASE_DESIGN.md §2 (full RBAC deferred, only two roles exist).

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(255) NOT NULL,
    password_hash   TEXT NOT NULL,
    full_name       VARCHAR(255) NOT NULL,
    role            VARCHAR(20) NOT NULL DEFAULT 'buyer',
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_users_role   CHECK (role IN ('buyer', 'admin')),
    CONSTRAINT chk_users_status CHECK (status IN ('active', 'suspended'))
);

-- Unique index on email, used for login lookups (DATABASE_DESIGN.md §2 "Indexes").
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (LOWER(email));

-- Index on status for admin queries (e.g. listing suspended accounts).
CREATE INDEX IF NOT EXISTS idx_users_status ON users (status);
