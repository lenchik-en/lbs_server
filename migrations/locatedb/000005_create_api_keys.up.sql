CREATE TABLE IF NOT EXISTS api_keys
(
    key        TEXT PRIMARY KEY,
    scope      TEXT        NOT NULL CHECK (scope IN ('locate', 'update', 'both')),
    is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
