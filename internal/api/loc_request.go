package api

const (
	Gsm   = "gsm"
	Wcdma = "wcdma"
	Lte   = "lte"
)

type LocateRequest struct {
	Cell []Cell `json:"cell"`
	Wifi []Wifi `json:"wifi"`
}

type Cell struct {
	Tech  string
	GSM   *GSM   `json:"gsm"`
	WCDMA *WCDMA `json:"wcdma"`
	LTE   *LTE   `json:"lte"`
}

type GSM struct {
	MCC            int `json:"mcc"`
	MNC            int `json:"mnc"`
	LAC            int `json:"lac"`
	CID            int `json:"cid"`
	SignalStrength int `json:"signal_strength"`
	BSIC           int `json:"bsic"`
	ARFCN          int `json:"arfcn"`
	Age            int `json:"age"`
	TimingAdvance  int `json:"timing_advance"`
}

type WCDMA struct {
	MCC            int `json:"mcc"`
	MNC            int `json:"MNC"`
	LAC            int `json:"lac"`
	CID            int `json:"cid"`
	SignalStrength int `json:"signal_strength"`
	PSC            int `json:"psc"`
	UARFCN         int `json:"uarfcn"`
	Age            int `json:"age"`
}

type LTE struct {
	MCC            int `json:"mcc"`
	MNC            int `json:"mnc"`
	TAC            int `json:"tac"`
	CI             int `json:"ci"`
	SignalStrength int `json:"signal_strength"`
	PCI            int `json:"pci"`
	EARFCN         int `json:"earfcn"`
	Age            int `json:"age"`
	TimingAdvance  int `json:"timing_advance"`
}

type Wifi struct {
	BSSID          int `json:"bssid"`
	SignalStrength int `json:"signal_strength"`
	CHANNEL        int `json:"channel"`
	AGE            int `json:"age"`
}
