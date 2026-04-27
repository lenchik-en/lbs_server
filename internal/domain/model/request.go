package model

type ObjectType string

const (
	ObjectTypeMetro    ObjectType = "metro"
	ObjectTypeBuilding ObjectType = "building"
	ObjectTypeStreet   ObjectType = "street"
)

type LocateRequest struct {
	SessionUUID string `json:"sessionUUID"`
	Cell        []Cell `json:"cell"`
	Wifi        []Wifi `json:"wifi"`
	IP          []Ip   `json:"ip"`
}

type LocateResponse struct {
	SessionUUID string    `json:"sessionUUID"`
	Location    *Location `json:"location"`
}

type UpdateRequest struct {
	SessionUUID string     `json:"sessionUUID,omitempty"`
	Type        ObjectType `json:"type,omitempty"`
	Cell        []Cell     `json:"cell,omitempty"`
	Wifi        []Wifi     `json:"wifi,omitempty"`
	IP          []Ip       `json:"ip,omitempty"`
	Location    *Location  `json:"location"`
}
