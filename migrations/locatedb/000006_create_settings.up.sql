CREATE TABLE IF NOT EXISTS server_settings
(
    id                   INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    require_key_locate   BOOLEAN NOT NULL DEFAULT FALSE,
    require_key_update   BOOLEAN NOT NULL DEFAULT FALSE,
    refresh_period_ms    INTEGER NOT NULL DEFAULT 0,
    locate_min_period_key  INTEGER NOT NULL DEFAULT 0,
    locate_min_period_anon INTEGER NOT NULL DEFAULT 0,
    update_min_period_key  INTEGER NOT NULL DEFAULT 0,
    update_min_period_anon INTEGER NOT NULL DEFAULT 0
);
