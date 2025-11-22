package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

/* key=<apiKey>&mcc=<mcc>&mnc=<mnc>&lac=<lac>&cellid=<cellid>&radio=<radio>&format=<format> */

type OpenCellRequest struct {
	Token string `json:"token"`
	Radio string `json:"radio"`
	Mcc   int    `json:"mcc"`
	Mnc   int    `json:"mnc"`
	Cells []struct {
		Lac int `json:"lac"`
		Cid int `json:"cid"`
		Psc int `json:"psc"`
	} `json:"cells"`
	Address int `json:"address"`
}

type OpenCellResponse struct {
	Status   string  `json:"status"`
	Balance  int     `json:"balance"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Accuracy int     `json:"accuracy"`
	Address  string  `json:"address"`
}

func Query(r OpenCellRequest) (*OpenCellResponse, error) {
	url := os.Getenv("OPENCELL_URL")
	if url == "" {
		return nil, fmt.Errorf("no url in OPENCELL_URL")
	}
	apiKey := os.Getenv("OPENCELL_API_KEY")
	if apiKey == "" {
		apiKey = "pk.643857ee95fe48606728636c03006e51"
	}
	r.Token = apiKey
	payload, _ := json.Marshal(r)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{Timeout: 2 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call OpenCellID: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	//fmt.Println("Raw response", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expected from OpenCellID 200 OK, got: %d: %s", resp.StatusCode, string(body))
	}

	var out OpenCellResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("failed to json decode responce body: %v", err)
	}

	return &out, nil
}
