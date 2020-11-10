package controllers

import (
	"app/application/response"
	"app/config"
	"app/domain/service"
	"net/http"
	"os"
	"strconv"
	"time"
)

// パラメータに応じた単位のローソク足情報を返す
func ApiCandleHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		productCode := r.URL.Query().Get("product_code")
		// パラメータで指定がない場合は設定ファイルのものを使う
		if productCode == "" {
			productCode = os.Getenv("PRODUCT_CODE")
		}
		strLimit := r.URL.Query().Get("limit")
		limit, err := strconv.Atoi(strLimit)
		if strLimit == "" || err != nil || limit < 0 || limit > 1000 {
			// デフォルトは1000とする
			limit = 1000
		}
		// 単位
		duration := r.URL.Query().Get("duration")
		if duration == "" {
			// デフォルトは分とする
			duration = "1m"
		}
		durationTime := config.Config.Durations[duration]

		df, _ := service.GetAllCandle(productCode, durationTime, limit)
		response.Success(w, df.Candles)
	}
}

func GetLatestCandle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		productCode := r.URL.Query().Get("product_code")
		// パラメータで指定がない場合は設定ファイルのものを使う
		if productCode == "" {
			productCode = os.Getenv("PRODUCT_CODE")
		}
		currentCandle := service.SelectOne(productCode, time.Minute, time.Now().Truncate(time.Minute))
		if currentCandle == nil {
			response.Success(w, time.Now())
		}
		response.Success(w, currentCandle)
	}
}
