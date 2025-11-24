package api

import (
	"os"
)

func ConvertLocateToOpenCell(req LocateRequest) OpenCellRequest {
	ocReq := OpenCellRequest{
		Token:   os.Getenv("OPENCELL_API_KEY"),
		Address: 0,
	}

	for _, c := range req.Cell {
		if c.GSM != nil {
			ocReq.Radio = Gsm
			ocReq.Mcc = c.GSM.MCC
			ocReq.Mnc = c.GSM.MNC
			ocReq.Cells = append(ocReq.Cells, struct {
				Lac int `json:"lac"`
				Cid int `json:"cid"`
				Psc int `json:"psc"`
			}{Lac: c.GSM.LAC, Cid: c.GSM.CID, Psc: 0})
		}
		if c.WCDMA != nil {
			ocReq.Radio = Wcdma
			ocReq.Mcc = c.WCDMA.MCC
			ocReq.Mnc = c.WCDMA.MNC
			ocReq.Cells = append(ocReq.Cells, struct {
				Lac int `json:"lac"`
				Cid int `json:"cid"`
				Psc int `json:"psc"`
			}{Lac: c.WCDMA.LAC, Cid: c.WCDMA.CID, Psc: c.WCDMA.PSC})
		}

		if c.LTE != nil {
			ocReq.Radio = Lte
			ocReq.Mcc = c.LTE.MCC
			ocReq.Mnc = c.LTE.MNC
			ocReq.Cells = append(ocReq.Cells, struct {
				Lac int `json:"lac"`
				Cid int `json:"cid"`
				Psc int `json:"psc"`
			}{Lac: c.LTE.TAC, Cid: c.LTE.CI, Psc: 0})
		}
	}

	return ocReq
}

func ConvertOpenCellToLocation(resp *OpenCellResponse) LocationResponse {
	var loc LocationResponse
	loc.Point.Lat = resp.Lat
	loc.Point.Lon = resp.Lon
	loc.Accuracy = resp.Accuracy
	return loc
}
