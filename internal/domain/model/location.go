package model

type Source string

const (
	SourceExternal Source = "external"
	SourceClient   Source = "client"
)

type Point struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Location struct {
	Point     Point  `json:"point"`
	Accuracy  int    `json:"accuracy"`
	Source    Source `json:"source,omitempty"`
	FromMetro bool   `json:"-"` // внутренний флаг, не отдаётся клиенту
}
