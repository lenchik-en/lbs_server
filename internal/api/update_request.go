package api

type UpdateRequest struct {
	Cell []Cell `json:"cell"`
	Wifi []Wifi `json:"wifi"`
	IP   []Ip   `json:"ip"`
	Location
}
