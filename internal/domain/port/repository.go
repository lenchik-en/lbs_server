package port

import (
	"context"

	"github.com/lenchik-en/lbs_server/internal/domain/model"
)

// LocationRepository — поиск координат в LocateDB.
type LocationRepository interface {
	FindLTE(ctx context.Context, lte *model.LTE) (*model.Location, error)
	FindGSM(ctx context.Context, gsm *model.GSM) (*model.Location, error)
	FindWCDMA(ctx context.Context, wcdma *model.WCDMA) (*model.Location, error)
	FindWifi(ctx context.Context, wifi *model.Wifi) (*model.Location, error)
	FindIP(ctx context.Context, ip *model.Ip) (*model.Location, error)
}

// ExternalRepository — поиск в ExternalDB (внешний датасет).
type ExternalRepository interface {
	FindLTE(ctx context.Context, lte *model.LTE) (*model.Location, error)
	FindGSM(ctx context.Context, gsm *model.GSM) (*model.Location, error)
	FindWCDMA(ctx context.Context, wcdma *model.WCDMA) (*model.Location, error)
}

// UpdateRepository — запись данных от клиентов в UpdateDB.
type UpdateRepository interface {
	InsertCell(ctx context.Context, cell *model.Cell, loc *model.Location, source, objectType string, rawJSON any) error
	InsertWifi(ctx context.Context, wifi *model.Wifi, loc *model.Location, source, objectType string, rawJSON any) error
	InsertIP(ctx context.Context, ip *model.Ip, loc *model.Location, source, objectType string, rawJSON any) error
}

// SessionRepository — управление сессиями и точками маршрута.
type SessionRepository interface {
	CreateIfNotExists(ctx context.Context, sessionUUID string) error
	SavePoint(ctx context.Context, sessionUUID string, lat, lon float64, accuracy int, source string, rawReq, rawResp any) error
}

// RefreshSource — постраничное чтение из UpdateDB для refresh.
type RefreshSource interface {
	FetchCellsForRefresh(offset, limit int) ([]model.CellRefreshRow, error)
	FetchWifisForRefresh(offset, limit int) ([]model.WifiRefreshRow, error)
	FetchIPsForRefresh(offset, limit int) ([]model.IPRefreshRow, error)
}

// RefreshDest — upsert в LocateDB при refresh.
type RefreshDest interface {
	UpsertCell(row model.CellRefreshRow) error
	UpsertWifi(row model.WifiRefreshRow) error
	UpsertIP(row model.IPRefreshRow) error
}

// APIKeyRepository — хранилище API-ключей.
type APIKeyRepository interface {
	GetByKey(ctx context.Context, key string) (*model.APIKey, error)
	Create(ctx context.Context, scope model.KeyScope) (*model.APIKey, error)
	Deactivate(ctx context.Context, key string) error
	ListAll(ctx context.Context) ([]*model.APIKey, error)
}
