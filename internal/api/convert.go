package api

import (
	"os"
)

func ConvertLocateToOpenCell(req LocateRequest) []OpenCellRequest {
	apiKey := os.Getenv("OPENCELL_API_KEY")

	var (
		gsmReq   OpenCellRequest
		wcdmaReq OpenCellRequest
		lteReq   OpenCellRequest
		results  []OpenCellRequest
	)

	gsmReq.Token = apiKey
	wcdmaReq.Token = apiKey
	lteReq.Token = apiKey

	for _, c := range req.Cell {
		if c.GSM != nil {
			gsmReq.Radio = Gsm
			gsmReq.Mcc = c.GSM.MCC
			gsmReq.Mnc = c.GSM.MNC
			gsmReq.Cells = append(gsmReq.Cells, struct {
				Lac int `json:"lac"`
				Cid int `json:"cid"`
				Psc int `json:"psc"`
			}{Lac: c.GSM.LAC, Cid: c.GSM.CID, Psc: 0})
		}
		if c.WCDMA != nil {
			wcdmaReq.Radio = Wcdma
			wcdmaReq.Mcc = c.WCDMA.MCC
			wcdmaReq.Mnc = c.WCDMA.MNC
			wcdmaReq.Cells = append(wcdmaReq.Cells, struct {
				Lac int `json:"lac"`
				Cid int `json:"cid"`
				Psc int `json:"psc"`
			}{Lac: c.WCDMA.LAC, Cid: c.WCDMA.CID, Psc: c.WCDMA.PSC})
		}

		if c.LTE != nil {
			lteReq.Radio = Lte
			lteReq.Mcc = c.LTE.MCC
			lteReq.Mnc = c.LTE.MNC
			lteReq.Cells = append(lteReq.Cells, struct {
				Lac int `json:"lac"`
				Cid int `json:"cid"`
				Psc int `json:"psc"`
			}{Lac: c.LTE.TAC, Cid: c.LTE.CI, Psc: 0})
		}
	}

	if len(lteReq.Cells) > 0 {
		results = append(results, lteReq)
	}
	if len(wcdmaReq.Cells) > 0 {
		results = append(results, wcdmaReq)
	}
	if len(gsmReq.Cells) > 0 {
		results = append(results, gsmReq)
	}

	return results
}

func ConvertOpenCellToLocation(resp *OpenCellResponse) LocationResponse {
	var loc LocationResponse
	loc.Point.Lat = resp.Lat
	loc.Point.Lon = resp.Lon
	loc.Accuracy = resp.Accuracy
	return loc
}
