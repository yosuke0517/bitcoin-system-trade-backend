package bitflyer

import (
	"encoding/json"
	"log"
)

// 証拠金情報
type Collateral struct {
	Collateral        float64 `json:"collateral"`
	OpenPositionPnl   float64 `json:"open_position_pnl"`
	RequireCollateral float64 `json:"require_collateral"`
	KeepRate          float64 `json:"keep_rate"`
}

/*
証拠金情報の取得
*/
func (api *APIClient) GetCollateral() (*Collateral, error) {
	url := "me/getcollateral"
	resp, _, err := api.doRequest("GET", url, map[string]string{}, nil)
	if err != nil {
		log.Printf("action=GetCollateral err=%s", err.Error())
		return nil, err
	}
	var collateral Collateral
	err = json.Unmarshal(resp, &collateral)
	if err != nil {
		log.Printf("action=GetCollateral err=%s", err.Error())
		return nil, err
	}
	return &collateral, nil
}
