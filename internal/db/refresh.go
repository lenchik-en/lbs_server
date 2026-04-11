package db

import "database/sql"

// CellRefreshRow — строка из UpdateDB для переноса в LocateDB.
type CellRefreshRow struct {
	ID               int64
	Tech             string
	MCC, MNC         int32
	LAC              sql.NullInt32
	TAC              sql.NullInt32
	CID              sql.NullInt32
	CI               sql.NullInt32
	PSC              sql.NullInt32
	PCI              sql.NullInt32
	ARFCN            sql.NullInt32
	EARFCN           sql.NullInt32
	UARFCN           sql.NullInt32
	BSIC             sql.NullInt32
	GSMTimingAdvance sql.NullInt32
	LTETimingAdvance sql.NullInt32
	GSMAge           sql.NullInt32
	WCDMAAge         sql.NullInt32
	LTEAge           sql.NullInt32
	Lat, Lon         float64
	ObjectType       sql.NullString
}

// WifiRefreshRow — строка из UpdateDB.wifi для переноса в LocateDB.
type WifiRefreshRow struct {
	BSSID    string
	Lat, Lon float64
}

// IPRefreshRow — строка из UpdateDB.ip для переноса в LocateDB.
type IPRefreshRow struct {
	Address  string
	Lat, Lon float64
}

// RefreshSource умеет постранично читать данные из UpdateDB для refresh.
type RefreshSource interface {
	FetchCellsForRefresh(offset, limit int) ([]CellRefreshRow, error)
	FetchWifisForRefresh(offset, limit int) ([]WifiRefreshRow, error)
	FetchIPsForRefresh(offset, limit int) ([]IPRefreshRow, error)
}

// RefreshDest умеет upsert-ить данные в LocateDB при refresh.
type RefreshDest interface {
	UpsertCell(row CellRefreshRow) error
	UpsertWifi(row WifiRefreshRow) error
	UpsertIP(row IPRefreshRow) error
}
