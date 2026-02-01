package api

type UpdateRequest struct {
	Cell     *Cell     `json:"cell,omitempty"`
	Wifi     *Wifi     `json:"wifi,omitempty"`
	IP       *Ip       `json:"ip,omitempty"`
	Location *Location `json:"location"`
}
