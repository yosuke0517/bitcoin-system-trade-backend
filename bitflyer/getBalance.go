package bitflyer

import (
	"encoding/json"
	"log"
)

/*
getBalanceのレスポンス
https://lightning.bitflyer.com/docs/playground#GETv1%2Fme%2Fgetbalance/javascript
*/
type Balance struct {
	CurrentCode string  `json:"current_code"`
	Amount      float64 `json:amount`
	Available   float64 `json:available`
}

/*
現在所持している現金やビットコインの情報を取得する
*/
func (api *APIClient) GetBalance() ([]Balance, error) {
	url := "me/getbalance"
	resp, _, err := api.doRequest("GET", url, map[string]string{}, nil)
	if err != nil {
		log.Printf("action=GetBalance err=%s", err.Error())
		return nil, err
	}
	var balance []Balance
	err = json.Unmarshal(resp, &balance)
	if err != nil {
		log.Printf("action=GetBalance err=%s", err.Error())
		return nil, err
	}
	return balance, nil
}
