CREATE TABLE IF NOT EXISTS sessions
(
    session_uuid UUID PRIMARY KEY, --идентификатор одной сессии
    started_at   TIMESTAMPTZ DEFAULT NOW(), --когда начали отправлять гео
    ended_at     TIMESTAMPTZ --когда закончили(!todo:уточнить, как понимать и надо ли вообще)
);

CREATE TABLE IF NOT EXISTS session_points
(
    id           BIGSERIAL PRIMARY KEY,
    session_uuid UUID REFERENCES sessions (session_uuid),
    timestamp    TIMESTAMPTZ DEFAULT NOW(),
    lat          DOUBLE PRECISION,
    lon          DOUBLE PRECISION,
    accuracy     INTEGER,
    source       TEXT,  -- "lte", "wifi", "gsm", "fusion", "cache"
    raw_request  JSONB, -- исходные input-башки от клиента
    raw_response JSONB
);

CREATE INDEX IF NOT EXISTS idx_session_points_session_ts ON session_points (session_uuid, timestamp);
