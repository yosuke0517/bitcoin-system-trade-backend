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
			duration = "5m"
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

		/** ema */
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

		/** ボリンジャーバンド */
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

		/** 一目均衡表 +/
		ichimoku := r.URL.Query().Get("ichimoku")
		if ichimoku != "" {
			df.AddIchimoku()
		}

		/** rsi */
		rsi := r.URL.Query().Get("rsi")
		if rsi != "" {
			strPeriod := r.URL.Query().Get("rsiPeriod")
			period, err := strconv.Atoi(strPeriod)
			// デフォルトは14
			if strPeriod == "" || err != nil || period < 0 {
				period = 14
			}
			df.AddRsi(period)
		}

		/** macd */
		macd := r.URL.Query().Get("macd")
		if macd != "" {
			strPeriod1 := r.URL.Query().Get("macdPeriod1")
			strPeriod2 := r.URL.Query().Get("macdPeriod2")
			strPeriod3 := r.URL.Query().Get("macdPeriod3")
			period1, err := strconv.Atoi(strPeriod1)
			if strPeriod1 == "" || err != nil || period1 < 0 {
				period1 = 12
			}
			period2, err := strconv.Atoi(strPeriod2)
			if strPeriod2 == "" || err != nil || period2 < 0 {
				period2 = 26
			}
			period3, err := strconv.Atoi(strPeriod3)
			if strPeriod3 == "" || err != nil || period3 < 0 {
				period3 = 9
			}
			df.AddMacd(period1, period2, period3)
		}

		/** ヒストリカルボラティリティ */
		hv := r.URL.Query().Get("hv")
		if hv != "" {
			strPeriod1 := r.URL.Query().Get("hvPeriod1")
			strPeriod2 := r.URL.Query().Get("hvPeriod2")
			strPeriod3 := r.URL.Query().Get("hvPeriod3")
			period1, err := strconv.Atoi(strPeriod1)
			if strPeriod1 == "" || err != nil || period1 < 0 {
				period1 = 21
			}
			period2, err := strconv.Atoi(strPeriod2)
			if strPeriod2 == "" || err != nil || period2 < 0 {
				period2 = 63
			}
			period3, err := strconv.Atoi(strPeriod3)
			if strPeriod3 == "" || err != nil || period3 < 0 {
				period3 = 252
			}
			df.AddHv(period1)
			df.AddHv(period2)
			df.AddHv(period3)
		}

		/** 売買イベント */
		events := r.URL.Query().Get("events")
		// TODO EMA一旦コメントアウト
		//if events != "" {
		//	if config.Config.BackTest {
		//		p, p1, p2 := df.OptimizeEma()
		//		log.Println(p, p1, p2)
		//		if p > 0 {
		//			df.Events = df.BackTestEma(p1, p2)
		//		} else {
		//			log.Println("利益が出ないため未実施")
		//		}
		//	} else {
		//		firstTime := df.Candles[0].Time
		//		df.AddEvents(firstTime)
		//	}
		//}

		// ボリンジャーバンド, 一目均衡表確認 TODO いったんコメントアウト
		//if events != "" {
		//	if config.Config.BackTest {
		//		//performance, p1, p2 := df.OptimizeBb()
		//		//log.Println(performance, p1, p2)
		//		//if performance > 0 {
		//		//	df.Events = df.BackTestBb(p1, p2)
		//		//}
		//		df.Events = df.BackTestIchimoku()
		//	} else {
		//		firstTime := df.Candles[0].Time
		//		df.AddEvents(firstTime)
		//	}
		//}

		// MACD確認
		//if events != "" {
		//	if config.Config.BackTest {
		//		performance, p1, p2, p3 := df.OptimizeMacd()
		//		log.Println(performance, p1, p2, p3)
		//		if performance > 0 {
		//			df.Events = df.BackTestMacd(p1, p2, p3)
		//		}
		//	} else {
		//		firstTime := df.Candles[0].Time
		//		df.AddEvents(firstTime)
		//	}
		//}

		if events != "" {
			if config.Config.BackTest {
				df.Events = Ai.SignalEvents.CollectAfter(df.Candles[0].Time)
			} else {
				firstTime := df.Candles[0].Time
				df.AddEvents(firstTime)
			}
		}

		// RSI確認用
		//if events != "" {
		//	if config.Config.BackTest {
		//		performance, p1, p2, p3 := df.OptimizeRsi()
		//		log.Println(performance, p1, p2, p3)
		//		if performance > 0 {
		//			df.Events = df.BackTestRsi(p1, p2, p3)
		//		}
		//	} else {
		//		firstTime := df.Candles[0].Time
		//		df.AddEvents(firstTime)
		//	}
		//}

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
