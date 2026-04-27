CREATE TABLE IF NOT EXISTS cells
(
    id       SERIAL PRIMARY KEY,

    radio    SMALLINT         NOT NULL, --GSM = 1, WCDMA = 2, LTE = 3, UMTS = 4
    mcc      INTEGER          NOT NULL, --Mobile Country Code
    mnc      INTEGER          NOT NULL, --Mobile Network Code

    area     INTEGER          NOT NULL, --LAC/TAC
    cell     BIGINT           NOT NULL, --CID/CI

    unit     INTEGER,                   --RNC / eNB / BS
    strength INTEGER,

    lat      DOUBLE PRECISION NOT NULL,
    lon      DOUBLE PRECISION NOT NULL,

    UNIQUE (radio, mcc, mnc, area, cell)
);

CREATE INDEX IF NOT EXISTS idx_cells_lookup
    ON cells (radio, mcc, mnc, area, cell);