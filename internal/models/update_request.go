package models

type Source string

const (
	EXTERNAL = "external"
	CLIENT   = "client"
)

type ObjectType string

const (
	ObjectTypeMetro    ObjectType = "metro"
	ObjectTypeBuilding ObjectType = "building"
	ObjectTypeStreet   ObjectType = "street"
)

type UpdateRequest struct {
	SessionUUID string     `json:"sessionUUID,omitempty"`
	Type        ObjectType `json:"type,omitempty"`
	Cell        []Cell     `json:"cell,omitempty"`
	Wifi        []Wifi     `json:"wifi,omitempty"`
	IP          []Ip       `json:"ip,omitempty"`
	Location    *Location  `json:"location"`
}
