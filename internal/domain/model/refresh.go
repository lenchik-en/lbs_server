package model

import "database/sql"

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
	Source           sql.NullString
}

type WifiRefreshRow struct {
	BSSID    string
	Lat, Lon float64
}

type IPRefreshRow struct {
	Address  string
	Lat, Lon float64
}

type RefreshStats struct {
	CellsUpserted int `json:"cells_upserted"`
	WifisUpserted int `json:"wifis_upserted"`
	IPsUpserted   int `json:"ips_upserted"`
}
