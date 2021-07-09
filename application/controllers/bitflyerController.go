package controllers

import (
	"app/bitflyer"
	"app/config"
	"app/domain/model"
	"app/domain/service"
	"log"
	"os"
	"strconv"
	"time"
)

var tradeTicker bitflyer.Ticker

var isTruncate bool
var isProduction, _ = strconv.ParseBool(os.Getenv("PRODUCTION"))

func StreamIngestionData() {
	ai := NewAI(os.Getenv("PRODUCT_CODE"), config.Config.Durations["15m"], config.Config.DataLimit, config.Config.UsePercent, config.Config.StopLimitPercent, config.Config.BackTest)

	var tickerChannl = make(chan bitflyer.Ticker)
	bitflyerClient := bitflyer.New(os.Getenv("API_KEY"), os.Getenv("API_SECRET"))
	go bitflyerClient.GetRealTimeTicker(os.Getenv("PRODUCT_CODE"), tickerChannl)
	go func() {
		for {
			for ticker := range tickerChannl {
				for _, duration := range config.Config.Durations {
					tradeTicker = ticker
					isCreated := service.CreateCandleWithDuration(ticker, ticker.ProductCode, duration)
					if isCreated == true {
					}
				}
			}
		}
	}()
	go func() {
		for range time.Tick(1 * time.Second) {
			eventLength := model.GetAllSignalEventsCount()
			df, _ := service.GetAllCandle(ai.ProductCode, ai.Duration, ai.PastPeriod)
			lenCandles := len(df.Candles)
			// キャンドル数が設定数ない場合取引しない
			candleLengthMin, _ := strconv.Atoi(os.Getenv("CANDLE_LENGTH_MIN"))
			if lenCandles < candleLengthMin {
				// めっっちゃログ出るからとりあえずコメントアウト
				//fmt.Printf("キャンドル数が設定値に満たないため取引しません。現在のキャンドル数: %s", strconv.Itoa(lenCandles))
				continue
			}
			if isProduction {
				if (time.Now().Hour() != 4 && time.Now().Second()%20 == 1000) || (time.Now().Hour() == 4 && eventLength%2 == 1 && time.Now().Second()%20 == 1000) {
					ai.Trade(tradeTicker)
				}
				if time.Now().Hour() == 4 && time.Now().Minute() == 0 && time.Now().Second() == 10 {
					if !isTruncate {
						isTruncate, _ = service.Truncate()
						if isTruncate {
							log.Println("テーブル削除完了")
							isTruncate = false
						}
					}
				}
			} else {
				if (time.Now().Hour() != 19 && time.Now().Second()%20 == 1000) || (time.Now().Hour() == 19 && eventLength%2 == 1 && time.Now().Second()%20 == 1000) {
					ai.Trade(tradeTicker)
				}
				if time.Now().Hour() == 19 && time.Now().Minute() == 0 && time.Now().Second() == 10 {
					if !isTruncate {
						isTruncate, _ = service.Truncate()
						if isTruncate {
							log.Println("テーブル削除完了")
							isTruncate = false
						}
					}
				}
			}
			// 取引時間6時~23時
			//if (time.Now().Hour() < 14 || time.Now().Hour() > 20) && time.Now().Second()%10 == 0 {
			//	ai.Trade(tradeTicker)
			//} else if time.Now().Hour() == 14 && time.Now().Minute() == 1 {
			//eventLength := model.GetAllSignalEventsCount()
			//	if eventLength%2 == 0 {
			//		// Truncate
			//		if !isTruncate {
			//			isTruncate, _ = service.Truncate()
			//			if isTruncate {
			//				log.Println("テーブル削除完了")
			//			}
			//		}
			//	}
			//}
			// Truncate フラグ初期化
			//if time.Now().Hour() == 20 && time.Now().Minute() == 59 {
			//	isTruncate = false
			//}
		}
	}()
}
