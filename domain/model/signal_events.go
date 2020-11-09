package model

import (
	"time"
)

// SignalEvents 売買のイベントを書き込む
type SignalEvents struct {
	Time        time.Time `json:"time"`
	ProductCode string    `json:"product_code"`
	Side        string    `json:"side"` // BUY or SELL
	Price       float64   `json:"price"`
	Size        float64   `json:"size"`
}
