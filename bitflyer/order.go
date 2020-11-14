package bitflyer

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// SendOrder 送るdata
type Order struct {
	ID                     int     `json:"id"`
	ChildOrderAcceptanceID string  `json:"child_order_acceptance_id"`
	ProductCode            string  `json:"product_code"`
	ChildOrderType         string  `json:"child_order_type"`
	Side                   string  `json:"side"`
	Price                  float64 `json:"price"`
	Size                   float64 `json:"size"`
	MinuteToExpires        int     `json:"minute_to_expire"`
	TimeInForce            string  `json:"time_in_force"`
	Status                 string  `json:"status"`
	ErrorMessage           string  `json:"error_message"`
	AveragePrice           float64 `json:"average_price"`
	ChildOrderState        string  `json:"child_order_state"`
	ExpireDate             string  `json:"expire_date"`
	ChildOrderDate         string  `json:"child_order_date"`
	OutstandingSize        float64 `json:"outstanding_size"`
	CancelSize             float64 `json:"cancel_size"`
	ExecutedSize           float64 `json:"executed_size"`
	TotalCommission        float64 `json:"total_commission"`
	Count                  int     `json:"count"`
	Before                 int     `json:"before"`
	After                  int     `json:"after"`
}

// SendOrder responce
type ResponseSendChildOrder struct {
	ChildOrderAcceptanceID string `json:"child_order_acceptance_id"`
}

// 注文を送る
func (api *APIClient) SendOrder(order *Order) (*ResponseSendChildOrder, error) {
	data, err := json.Marshal(order)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	url := "me/sendchildorder"
	resp, _, err := api.doRequest("POST", url, map[string]string{}, data)
	fmt.Println(resp)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	var response ResponseSendChildOrder
	err = json.Unmarshal(resp, &response)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return &response, nil
}

// 注文の詳細を取得する
func (api *APIClient) ListOrder(query map[string]string) ([]Order, error) {
	resp, _, err := api.doRequest("GET", "me/getchildorders", query, nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	var responseListOrder []Order
	err = json.Unmarshal(resp, &responseListOrder)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return responseListOrder, nil
}

// キャンセルStruct
type CancelOrder struct {
	ProductCode            string `json:"product_code"`
	ChildOrderAcceptanceID string `json:"child_order_acceptance_id"`
}

// オーダーをキャンセルする
func (api *APIClient) CancelOrder(cancelOrder *CancelOrder) (int, error) {
	data, err := json.Marshal(cancelOrder)
	if err != nil {
		return 400, err
	}
	url := "me/cancelchildorder"
	_, statusCode, err := api.doRequest("POST", url, map[string]string{}, data)
	if err != nil {
		return 400, err
	}
	return statusCode, err
}

/*
データベースが対応している日付型になおすメソッド
*/
func (o *Order) DateTime() time.Time {
	layout := "2006-01-02T15:04:05"
	dateTime, err := time.Parse(layout, o.ChildOrderDate)
	if err != nil {
		log.Printf("action=DateTime, err=%s", err.Error())
	}
	return dateTime
}

/*
時間変換用メソッド
@param 時間単位：h,m,s
@return time.Time（12:12:00 → duration hを与えると12:00:00に変換される
*/
func (o *Order) TruncateDateTime(duration time.Duration) time.Time {
	return o.DateTime().Truncate(duration)
}
