package controllers

import (
	"app/bitflyer"
	"app/config"
	"app/domain/service"
	"os"
)

func StreamIngestionData() {
	// ai := NewAI(os.Getenv("PRODUCT_CODE"), config.Config.Durations["1m"], config.Config.DataLimit, config.Config.UsePercent, config.Config.StopLimitPercent, config.Config.BackTest)

	var tickerChannl = make(chan bitflyer.Ticker)
	bitflyerClient := bitflyer.New(os.Getenv("API_KEY"), os.Getenv("API_SECRET"))
	go bitflyerClient.GetRealTimeTicker(os.Getenv("PRODUCT_CODE"), tickerChannl)
	go func() {
		for {
			for ticker := range tickerChannl {
				for _, duration := range config.Config.Durations {
					isCreated := service.CreateCandleWithDuration(ticker, ticker.ProductCode, duration)
					if isCreated == true && duration == config.Config.Durations["1m"] {
						// ai.Trade()
					}
				}
			}
		}
	}()
}
