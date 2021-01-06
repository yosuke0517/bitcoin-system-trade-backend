package service

import (
	"app/bitflyer"
	"app/config"
	"app/domain/model"
	"github.com/markcheno/go-talib"
	"time"
)

// キャンドル情報を保存する
func CreateCandleWithDuration(ticker bitflyer.Ticker, productCode string, duration time.Duration) bool {
	currentCandle := SelectOne(productCode, duration, ticker.TruncateDateTime(duration))
	price := ticker.GetMidPrice()
	// 秒単位は毎回insert
	if currentCandle == nil {
		candle := NewCandle(productCode, duration, ticker.TruncateDateTime(duration),
			price, price, price, price, ticker.Volume)
		if candle != nil {
		}
		candle.Insert()
		return true
	}
	// High, Lowの更新があった場合のみデータベースに保存する
	shouldSave := false
	// 分・時単位は秒単位ではupdateする
	if currentCandle.High < price {
		currentCandle.High = price
		shouldSave = true
	} else if currentCandle.Low > price {
		currentCandle.Low = price
		shouldSave = true
	}
	// Volumeは毎回足す
	currentCandle.Volume += ticker.Volume
	if shouldSave == true || time.Now().Truncate(time.Second).Second() > 59 {
		currentCandle.Close = price
		currentCandle.Save()
		shouldSave = false
	}
	return false
}

// chart?product_code=FX_BTC_JPY&duration=1h
func GetAllCandle(productCode string, duration time.Duration, limit int) (dfCandle *model.DataFrameCandle, err error) {
	rows := SelectAll(productCode, duration, limit)
	defer rows.Close()
	dfCandle = &model.DataFrameCandle{}
	dfCandle.ProductCode = productCode
	dfCandle.Duration = duration
	if rows == nil {
		return
	}
	for rows.Next() {
		var candle model.Candle
		candle.ProductCode = productCode
		candle.Duration = duration
		rows.Scan(&candle.Time, &candle.Open, &candle.Close, &candle.High, &candle.Low, &candle.Volume)
		dfCandle.Candles = append(dfCandle.Candles, candle)
	}
	err = rows.Err()
	if err != nil {
		return
	}
	return dfCandle, nil
}

// 指定された分数分のVolumeを取得する
func IsAvailableVolume() bool {
	dfH, err := GetAllCandle(config.Config.ProductCode, time.Duration(time.Hour), 2)
	if err != nil {
		return false
	}
	if len(dfH.Candles) == 2 {
		if int(dfH.Candles[1].Volume) > 10000000 {
			return true
		}
	}
	return false
}

// 指定された分のキャンドルに対するボラティリティを出力する
func Atr(limit int) (int, error) {
	dfM, err := GetAllCandle(config.Config.ProductCode, time.Duration(time.Minute), limit)
	if err != nil {
		return 0, err
	}
	if len(dfM.Highs()) == limit {
		atr := talib.Atr(dfM.Highs(), dfM.Low(), dfM.Closes(), limit-1)
		return int(atr[len(atr)-1]), nil
	}
	return 0, nil
}

func Obv(limit int) {
	//dfM, err := GetAllCandle(config.Config.ProductCode, time.Duration(time.Minute), limit)
}
