-- Auth DB: refresh_tokens table.
--
-- Design note: per docs/decisions/ADR-007-redis.md, Redis is the source of
-- truth for *active* refresh-token session validity (fast lookup/TTL/O(1)
-- revocation on logout). This Postgres table is a durable, queryable
-- record of every refresh token ever issued/rotated/revoked, used for:
--   - security audit trail (DATABASE_DESIGN.md §2 "Audit Requirements":
--     login/password/role changes must be auditable),
--   - recovering session history after a Redis restart without
--     persistence (ADR-007 "Negatif": Redis data loss should only cost a
--     re-login, never business data - this table is what makes that true
--     for the audit trail specifically),
--   - detecting refresh-token reuse (a classic sign of token theft: a
--     rotated/revoked token being replayed).
--
-- Postgres is therefore NOT queried on the hot path (every request); it is
-- written to on issue/rotate/revoke and read for audits/investigations.

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    issued_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked     BOOLEAN NOT NULL DEFAULT FALSE,
    revoked_at  TIMESTAMPTZ,
    replaced_by UUID, -- points to the new token id after rotation, for reuse-detection

    CONSTRAINT chk_refresh_tokens_expiry CHECK (expires_at > issued_at)
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens (expires_at);
