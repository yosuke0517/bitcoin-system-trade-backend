package bitflyer

import (
	"encoding/json"
	"log"
)

// GetTradingCommission responce
type TradingCommission struct {
	CommissionRate float64 `json:"commission_rate"`
}

// get GetTradingCommission 手数料を取得する
func (api *APIClient) GetTradingCommission(productCode string) (*TradingCommission, error) {
	url := "me/gettradingcommission"
	resp, _, err := api.doRequest("GET", url, map[string]string{"product_code": productCode}, nil)
	if err != nil {
		log.Printf("action=GetTradingCommission err=%s", err.Error())
		return nil, err
	}
	var tradingCommission TradingCommission
	err = json.Unmarshal(resp, &tradingCommission)
	if err != nil {
		log.Printf("action=GetTradingCommission err=%s", err.Error())
		return nil, err
	}
	return &tradingCommission, nil
}
