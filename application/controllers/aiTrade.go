package controllers

import (
	"app/bitflyer"
	"app/config"
	"app/domain/model"
	"app/domain/service"
	"app/domain/tradingalgo"
	"fmt"
	"github.com/markcheno/go-talib"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"golang.org/x/sync/semaphore"
)

type AI struct {
	API                  *bitflyer.APIClient
	ProductCode          string
	CurrencyCode         string
	CoinCode             string
	UsePercent           float64
	MinuteToExpires      int
	Duration             time.Duration
	PastPeriod           int
	SignalEvents         *model.SignalEvents
	OptimizedTradeParams *model.TradeParams
	TradeSemaphore       *semaphore.Weighted
	StopLimit            float64
	StopLimitPercent     float64
	BackTest             bool
	StartTrade           time.Time
}

// TODO mutex, singleton
var Ai *AI

func NewAI(productCode string, duration time.Duration, pastPeriod int, UsePercent, stopLimitPercent float64, backTest bool) *AI {
	apiClient := bitflyer.New(os.Getenv("API_KEY"), os.Getenv("API_SECRET"))
	var signalEvents *model.SignalEvents
	signalEvents = model.GetSignalEventsByCount(1, config.Config.BackTest)
	codes := strings.Split(productCode, "_")
	Ai = &AI{
		API:              apiClient,
		ProductCode:      productCode,
		CoinCode:         codes[2],
		CurrencyCode:     codes[1],
		UsePercent:       UsePercent,
		MinuteToExpires:  1, // どれくらいオーダーを保持するか（単位：分）
		PastPeriod:       pastPeriod,
		Duration:         duration,
		SignalEvents:     signalEvents,
		TradeSemaphore:   semaphore.NewWeighted(1), // AIでのトレード中は他のgorutineはできないようにする
		BackTest:         backTest,
		StartTrade:       time.Now(),
		StopLimitPercent: stopLimitPercent,
	}
	Ai.UpdateOptimizeParams(false)
	return Ai
}

/** インディケータの最適化 */
func (ai *AI) UpdateOptimizeParams(isContinue bool) {
	df, _ := service.GetAllCandle(ai.ProductCode, ai.Duration, ai.PastPeriod)
	ai.OptimizedTradeParams = df.OptimizeParams()
	log.Printf("optimized_trade_params=%+v", ai.OptimizedTradeParams)
	// インディケータが1つも使えない場合は再起呼び出し
	if ai.OptimizedTradeParams == nil && isContinue && !ai.BackTest {
		log.Print("status_no_params")
		time.Sleep(10 * ai.Duration)
		ai.UpdateOptimizeParams(isContinue)
	}
}

/** bitflyer用のBUY */
func (ai *AI) Buy(candle model.Candle) (childOrderAcceptanceID string, isOrderCompleted bool) {
	if ai.BackTest {
		couldBuy := ai.SignalEvents.Buy(ai.ProductCode, candle.Time, candle.Close, 1.0, true)
		return "", couldBuy
	}
	// トレード時間の妥当性チェック
	if ai.StartTrade.After(candle.Time) {
		return
	}

	if !ai.SignalEvents.CanBuy(candle.Time) {
		return
	}
	// 使用できる証拠金と取引中かどうかの判定
	availableCurrency := ai.GetAvailableBalance()
	// 使用して良い金額は証拠金に3.7をかけた数とする
	useCurrency := availableCurrency * ai.UsePercent
	ticker, err := ai.API.GetTicker(ai.ProductCode)
	if err != nil {
		log.Println(err)
		return
	}
	// 証拠金の4倍でどれだけ買えるか調査
	size := 1.0 / (ticker.BestAsk / useCurrency)
	size = ai.AdjustSize(size)

	params := map[string]string{
		"product_code": "FX_BTC_JPY",
	}
	positionRes, _ := ai.API.GetPositions(params)
	// positionResが1以上の場合、注文を決済するのでSizeを格納する
	if len(positionRes) != 0 {
		size = positionRes[0].Size
	}

	order := &bitflyer.Order{
		ProductCode:     ai.ProductCode,
		ChildOrderType:  "MARKET",
		Side:            "BUY",
		Size:            size,
		MinuteToExpires: ai.MinuteToExpires,
		TimeInForce:     "GTC",
	}
	log.Printf("status=order candle=%+v order=%+v", candle, order)
	resp, err := ai.API.SendOrder(order)
	if err != nil {
		log.Println(err)
		return
	}
	childOrderAcceptanceID = resp.ChildOrderAcceptanceID
	if resp.ChildOrderAcceptanceID == "" {
		// Insufficient fund
		// 資金が足りなくて買えない時もここに入ってくる
		log.Printf("order=%+v status=no_id", order)
		return
	}

	isOrderCompleted = ai.WaitUntilOrderComplete(childOrderAcceptanceID, candle.Time)
	return childOrderAcceptanceID, isOrderCompleted
}

/** bitflyer用のSELL */
func (ai *AI) Sell(candle model.Candle) (childOrderAcceptanceID string, isOrderCompleted bool) {
	if ai.BackTest {
		couldSell := ai.SignalEvents.Sell(ai.ProductCode, candle.Time, candle.Close, 1.0, true)
		return "", couldSell
	}

	if ai.StartTrade.After(candle.Time) {
		return
	}

	if !ai.SignalEvents.CanSell(candle.Time) {
		return
	}

	// 使用できる証拠金と取引中かどうかの判定
	availableCurrency := ai.GetAvailableBalance()
	// 使用して良い金額は証拠金に3.7をかけた数とする
	useCurrency := availableCurrency * ai.UsePercent
	ticker, err := ai.API.GetTicker(ai.ProductCode)
	if err != nil {
		log.Println(err)
		return
	}
	// 証拠金の4倍でどれだけ買えるか調査
	size := 1.0 / (ticker.BestAsk / useCurrency)
	size = ai.AdjustSize(size)

	params := map[string]string{
		"product_code": "FX_BTC_JPY",
	}
	positionRes, _ := ai.API.GetPositions(params)
	fmt.Println("きてる？？")
	// positionResが1以上の場合、注文を決済するのでSizeを格納する
	if len(positionRes) != 0 {
		size = positionRes[0].Size
	}

	order := &bitflyer.Order{
		ProductCode:     ai.ProductCode,
		ChildOrderType:  "MARKET",
		Side:            "SELL",
		Size:            size,
		MinuteToExpires: ai.MinuteToExpires,
		TimeInForce:     "GTC",
	}
	log.Printf("status=sell candle=%+v order=%+v", candle, order)
	resp, err := ai.API.SendOrder(order)
	if err != nil {
		log.Println(err)
		return
	}
	if resp.ChildOrderAcceptanceID == "" {
		// Insufficient funds
		log.Printf("order=%+v status=no_id", order)
		return
	}
	childOrderAcceptanceID = resp.ChildOrderAcceptanceID
	isOrderCompleted = ai.WaitUntilOrderComplete(childOrderAcceptanceID, candle.Time)
	return childOrderAcceptanceID, isOrderCompleted
}

func (ai *AI) Trade() {
	eventLength := model.GetAllSignalEventsCount()
	fmt.Println(eventLength)
	//if eventLength%2 == 0 {
	//	go ai.UpdateOptimizeParams(true)
	//}
	// goroutineの同時実行数を制御
	isAcquire := ai.TradeSemaphore.TryAcquire(1)
	if !isAcquire {
		log.Println("Could not get trade lock")
		return
	}
	defer ai.TradeSemaphore.Release(1)
	params := ai.OptimizedTradeParams
	log.Println(params)
	if params == nil {
		return
	}
	df, _ := service.GetAllCandle(ai.ProductCode, ai.Duration, ai.PastPeriod)
	lenCandles := len(df.Candles)

	// EMA
	var emaValues1 []float64
	var emaValues2 []float64
	if params.EmaEnable {
		emaValues1 = talib.Ema(df.Closes(), params.EmaPeriod1)
		emaValues2 = talib.Ema(df.Closes(), params.EmaPeriod2)
	}

	// ボリンジャーバンド
	var bbUp []float64
	var bbDown []float64
	if params.BbEnable {
		bbUp, _, bbDown = talib.BBands(df.Closes(), params.BbN, params.BbK, params.BbK, 0)
	}

	// 一目均衡表
	var tenkan, kijun, senkouA, senkouB, chikou []float64
	if params.IchimokuEnable {
		tenkan, kijun, senkouA, senkouB, chikou = tradingalgo.IchimokuCloud(df.Closes())
	}

	// MACD
	var outMACD, outMACDSignal []float64
	if params.MacdEnable {
		outMACD, outMACDSignal, _ = talib.Macd(df.Closes(), params.MacdFastPeriod, params.MacdSlowPeriod, params.MacdSignalPeriod)
	}

	// RSI
	var rsiValues []float64
	if params.RsiEnable {
		rsiValues = talib.Rsi(df.Closes(), params.RsiPeriod)
	}

	for i := 1; i < lenCandles; i++ {
		// 有効なインディケータの数
		buyPoint, sellPoint := 0, 0
		// ゴールデンクロス・デッドクロスが計算できる条件
		if params.EmaEnable && params.EmaPeriod1 <= i && params.EmaPeriod2 <= i {
			// ゴールデンクロス TODO 条件を追加すればさらに確度の高いトレードができる ex...df.Volume()[i] > 100とか
			if emaValues1[i-1] < emaValues2[i-1] && emaValues1[i] >= emaValues2[i] {
				buyPoint++
			}
			// デッドクロス
			if emaValues1[i-1] > emaValues2[i-1] && emaValues1[i] <= emaValues2[i] {
				sellPoint++
			}
		}

		// ボリンジャーバンド
		if params.BbEnable && params.BbN <= i {
			// 上抜け（買い）
			if bbDown[i-1] > df.Candles[i-1].Close && bbDown[i] <= df.Candles[i].Close {
				buyPoint++
			}
			// 下抜け（売り）
			if bbUp[i-1] < df.Candles[i-1].Close && bbUp[i] >= df.Candles[i].Close {
				sellPoint++
			}
		}

		// MACD
		if params.MacdEnable {
			// 上抜け（買い）
			if outMACD[i] < 0 && outMACDSignal[i] < 0 && outMACD[i-1] < outMACDSignal[i-1] && outMACD[i] >= outMACDSignal[i] {
				buyPoint++
			}
			// 下抜け（売り）
			if outMACD[i] > 0 && outMACDSignal[i] > 0 && outMACD[i-1] > outMACDSignal[i-1] && outMACD[i] <= outMACDSignal[i] {
				sellPoint++
			}
		}
		// 一目均衡表
		if params.IchimokuEnable {
			if chikou[i-1] < df.Candles[i-1].High && chikou[i] >= df.Candles[i].High &&
				senkouA[i] < df.Candles[i].Low && senkouB[i] < df.Candles[i].Low &&
				tenkan[i] > kijun[i] {
				buyPoint++
			}

			if chikou[i-1] > df.Candles[i-1].Low && chikou[i] <= df.Candles[i].Low &&
				senkouA[i] > df.Candles[i].High && senkouB[i] > df.Candles[i].High &&
				tenkan[i] < kijun[i] {
				sellPoint++
			}
		}
		// RSI
		if params.RsiEnable && rsiValues[i-1] != 0 && rsiValues[i-1] != 100 {
			// 30% 上抜け（買い）
			if rsiValues[i-1] < params.RsiBuyThread && rsiValues[i] >= params.RsiBuyThread {
				buyPoint++
			}
			// 70% 下抜け（売り）
			if rsiValues[i-1] > params.RsiSellThread && rsiValues[i] <= params.RsiSellThread {
				sellPoint++
			}
		}
		// 1つでも買いのインディケータがあれば買い TODO イジる
		//if buyPoint > 0 || ai.StopLimit < df.Candles[i].Close {
		//	_, isOrderCompleted := ai.Buy(df.Candles[i])
		//	if !isOrderCompleted {
		//		continue
		//	}
		//	// 順張りトレード
		//	// ストップリミット(× 90%）
		//	if eventLength%2 == 0 {
		//		// ストップリミット時はストップリミット初期化とインディケータの最適化
		//		ai.StopLimit = 0.0
		//		go ai.UpdateOptimizeParams(true)
		//	} else {
		//		ai.StopLimit = df.Candles[i].Close * ai.StopLimitPercent
		//	}
		//}
		//// 空売り
		//if sellPoint > 0 || ai.StopLimit > df.Candles[i].Close {
		//	_, isOrderCompleted := ai.Sell(df.Candles[i])
		//	if !isOrderCompleted {
		//		continue
		//	}
		//	// ストップリミット(× 110%）
		//	if eventLength%2 == 0 {
		//		// ストップリミット時はストップリミット初期化とインディケータの最適化
		//		ai.StopLimit = 0.0
		//		go ai.UpdateOptimizeParams(true)
		//	} else {
		//		ai.StopLimit = df.Candles[i].Close * (1.0 + (1.0 - ai.StopLimitPercent))
		//	}
		//}
		if buyPoint > 0 {
			_, isOrderCompleted := ai.Buy(df.Candles[i])
			if !isOrderCompleted {
				continue
			}
			ai.StopLimit = df.Candles[i].Close * ai.StopLimitPercent
		}

		if sellPoint > 0 || ai.StopLimit > df.Candles[i].Close {
			_, isOrderCompleted := ai.Sell(df.Candles[i])
			if !isOrderCompleted {
				continue
			}
			ai.StopLimit = 0.0
			ai.UpdateOptimizeParams(true)
		}
	}
}

/** 使用できる証拠金と取引中かどうかを返す
availableCurrency: 使用可能な証拠金
isTrading: 取引中かどうか
*/
func (ai *AI) GetAvailableBalance() (availableCurrency float64) {
	//isTrading = false
	balances, err := ai.API.GetCollateral()
	if err != nil {
		log.Println(err)
		return
	}
	return balances.Collateral
}

/** 購入・売却できるビットコインの数量を返す */
func (ai *AI) AdjustSize(size float64) float64 {
	return math.Floor(size*10000) / 10000
}

/** 注文が確定したかを確認し、signalEventsテーブルに売買情報を保存する */
func (ai *AI) WaitUntilOrderComplete(childOrderAcceptanceID string, executeTime time.Time) bool {
	params := map[string]string{
		"product_code":              ai.ProductCode,
		"child_order_acceptance_id": childOrderAcceptanceID,
	}
	expire := time.After(time.Minute)
	interval := time.Tick(15 * time.Second)
	return func() bool {
		for {
			select {
			case <-expire:
				return false
			case <-interval:
				listOrders, err := ai.API.ListOrder(params)
				if err != nil {
					log.Println(err)
					return false
				}
				if len(listOrders) == 0 {
					return false
				}
				order := listOrders[0]
				if order.ChildOrderState == "COMPLETED" {
					if order.Side == "BUY" {
						couldBuy := ai.SignalEvents.Buy(ai.ProductCode, executeTime, order.AveragePrice, order.Size, true)
						if !couldBuy {
							log.Printf("status=buy childOrderAcceptanceID=%s order=%+v", childOrderAcceptanceID, order)
						}
						return couldBuy
					}
					if order.Side == "SELL" {
						couldSell := ai.SignalEvents.Sell(ai.ProductCode, executeTime, order.AveragePrice, order.Size, true)
						if !couldSell {
							log.Printf("status=sell childOrderAcceptanceID=%s order=%+v", childOrderAcceptanceID, order)
						}
						return couldSell
					}
					return false
				}
			}
		}
	}()
}
