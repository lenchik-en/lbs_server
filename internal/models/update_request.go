package models

type UpdateRequest struct {
	SessionUUID string    `json:"sessionUUID,omitempty"`
	Cell        []Cell    `json:"cell,omitempty"`
	Wifi        []Wifi    `json:"wifi,omitempty"`
	IP          []Ip      `json:"ip,omitempty"`
	Location    *Location `json:"location"`
}
