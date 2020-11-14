package bitflyer

import (
	"encoding/json"
	"fmt"
	"log"
)

type Position struct {
	ProductCode         string  `json:"product_code"`
	Side                string  `json:"side"`
	Price               float64 `json:"price"`
	Size                float64 `json:"size"`
	Commission          float64 `json:"commission"`
	SwapPointAccumulate float64 `json:"swap_point_accumulate"`
	RequireCollateral   float64 `json:"require_collateral"`
	OpenDate            string  `json:"open_date"`
	Leverage            float64 `json:"leverage"`
	Pnl                 float64 `json:"pnl"`
	Sfd                 float64 `json:"sfd"`
}

// 建玉を取得する
func (api *APIClient) GetPositions(query map[string]string) ([]Position, error) {
	resp, _, err := api.doRequest("GET", "me/getpositions", query, nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	var position []Position
	err = json.Unmarshal(resp, &position)
	if err != nil {
		fmt.Printf("err: %s", err)
		return nil, err
	}
	return position, nil
}
