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

		// 単純移動平均線
		sma := r.URL.Query().Get("sma")
		if sma != "" {
			// パラメータが入っていない場合、デフォルトは7,14,50
			strSmaPeriod1 := r.URL.Query().Get("smaPeriod1")
			strSmaPeriod2 := r.URL.Query().Get("smaPeriod2")
			strSmaPeriod3 := r.URL.Query().Get("smaPeriod3")
			period1, err := strconv.Atoi(strSmaPeriod1)
			if strSmaPeriod1 == "" || err != nil || period1 < 0 {
				period1 = 7
			}
			period2, err := strconv.Atoi(strSmaPeriod2)
			if strSmaPeriod2 == "" || err != nil || period2 < 0 {
				period2 = 14
			}
			period3, err := strconv.Atoi(strSmaPeriod3)
			if strSmaPeriod3 == "" || err != nil || period3 < 0 {
				period3 = 50
			}
			df.AddSma(period1)
			df.AddSma(period2)
			df.AddSma(period3)
		}

		// 指数平滑移動平均線
		ema := r.URL.Query().Get("ema")
		if ema != "" {
			strEmaPeriod1 := r.URL.Query().Get("emaPeriod1")
			strEmaPeriod2 := r.URL.Query().Get("emaPeriod2")
			strEmaPeriod3 := r.URL.Query().Get("emaPeriod3")
			period1, err := strconv.Atoi(strEmaPeriod1)
			if strEmaPeriod1 == "" || err != nil || period1 < 0 {
				period1 = 7
			}
			period2, err := strconv.Atoi(strEmaPeriod2)
			if strEmaPeriod2 == "" || err != nil || period2 < 0 {
				period2 = 14
			}
			period3, err := strconv.Atoi(strEmaPeriod3)
			if strEmaPeriod3 == "" || err != nil || period3 < 0 {
				period3 = 50
			}
			df.AddEma(period1)
			df.AddEma(period2)
			df.AddEma(period3)
		}

		bbands := r.URL.Query().Get("bbands")
		if bbands != "" {
			strN := r.URL.Query().Get("bbandsN")
			strK := r.URL.Query().Get("bbandsK")
			n, err := strconv.Atoi(strN)
			if strN == "" || err != nil || n < 0 {
				n = 20
			}
			k, err := strconv.Atoi(strK)
			if strK == "" || err != nil || k < 0 {
				k = 2
			}
			df.AddBBands(n, float64(k))
		}

		response.Success(w, df)
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
