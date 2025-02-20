package controllers

import (
	"app/bitflyer"
	"app/config"
	"app/domain/model"
	"app/domain/service"
	"app/utils"
	"fmt"
	"github.com/markcheno/go-talib"
	"log"
	"math"
	"strconv"
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
	Profit               float64
}

// TODO mutex, singleton
var Ai *AI

var longReOpen bool

var shortReOpen bool

var tradeDuration int

func NewAI(productCode string, duration time.Duration, pastPeriod int, UsePercent, stopLimitPercent float64, backTest bool) *AI {
	apiClient := bitflyer.New(config.Config.ApiKey, config.Config.ApiSecret)
	tradeDuration, _ = strconv.Atoi(strings.TrimSuffix(config.Config.TradeDuration, config.Config.TradeSuffix))
	var signalEvents *model.SignalEvents
	signalEvents = model.GetSignalEventsByCount(1)
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
	Ai.UpdateOptimizeParams(false, false)
	return Ai
}

/** インディケータの最適化 */
func (ai *AI) UpdateOptimizeParams(isContinue, reOpen bool) {
	df, _ := service.GetAllCandle(ai.ProductCode, ai.Duration, ai.PastPeriod)
	ai.OptimizedTradeParams = df.OptimizeParams(reOpen)
	log.Printf("optimized_trade_params=%+v", ai.OptimizedTradeParams)
	// インディケータが1つも使えない場合は再起呼び出し
	if ai.OptimizedTradeParams == nil && isContinue && !ai.BackTest {
		log.Print("status_no_params")
		reOpen = false
		if longReOpen || shortReOpen {
			reOpen = true
		}
		ai.UpdateOptimizeParams(isContinue, reOpen)
	}
}

/** bitflyer用のBUY */
func (ai *AI) Buy(candle model.Candle, price, bbRate float64) (childOrderAcceptanceID string, isOrderCompleted bool, orderPrice float64) {
	orderPrice = 0.0
	pnl := 0.0
	size = 0.0
	atr, _ := service.Atr(30)
	// トレード時間の妥当性チェック
	if ai.StartTrade.After(candle.Time) {
		log.Println("candle.TimeがStartTradeより過去のため取引しません")
		return "timeError", false, 0.0
	}
	// ショートの利益確定後にロングでインしないようにreOpenをfalseにする
	if isShortProfit {
		longReOpen = false
	}
	if !ai.SignalEvents.CanBuy(candle.Time, longReOpen) {
		return
	}

	if !ai.BackTest {
		availableCurrency := ai.GetAvailableBalance()
		// 使用して良い金額は証拠金に3.5をかけた数とする
		useCurrency := availableCurrency * ai.UsePercent
		ticker, err := ai.API.GetTicker(ai.ProductCode)
		if err != nil {
			log.Println(err)
			return
		}
		if ticker == nil {
			return
		}
		// 証拠金の4倍でどれだけ買えるか調査
		size = 1.0 / (ticker.BestAsk / useCurrency)
		size = ai.AdjustSize(size)

		params := map[string]string{
			"product_code": "FX_BTC_JPY",
		}
		positionRes, _ := ai.API.GetPositions(params)
		fmt.Println("positionRessssssssssss")
		fmt.Println(positionRes)
		// positionResの中身
		// (注文単位で配列で返却される)positionResが1以上の場合、注文を決済するのでSizeを格納する
		if len(positionRes) > 0 {
			// positionResがあった場合、sizeを初期化（上記で新規購入のsizeを出しているので決済のsizeで上書きする）
			size = 0.0
			for _, position := range positionRes {
				size += position.Size
				pnl += position.Pnl
			}
			size = math.Round(size*10000) / 10000
		}
		if math.IsNaN(size) {
			log.Println("sizeの計算が出来ませんでした。BUYを中止します。")
			return
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
			log.Printf("order=%+v status=no_id（不正なsize指定がされている可能性があります。）", order)
			return
		}
		isOrderCompleted, orderPrice = ai.WaitUntilOrderComplete(childOrderAcceptanceID, pnl, bbRate)
		// continueフラグがtrueのときは連続売買する。positionResが0件のときは新規なのでReOpenはしない
		if config.Config.Continue && len(positionRes) > 0 && !isShortProfit {
			longReOpen = true
		}
		// StopLimit後はreOpenしない
		if (config.Config.Continue && isStopLimit) || !config.Config.Continue {
			longReOpen = false
		}
		return childOrderAcceptanceID, isOrderCompleted, orderPrice
	} else {
		couldBuy := ai.SignalEvents.Buy(ai.ProductCode, time.Now(), candle.Close, 1.0, true, longReOpen, price, atr, pnl, bbRate)
		utils.SendLine("couldBuy： " + strconv.FormatBool(couldBuy))
		return "", couldBuy, candle.Close
	}
}

/** bitflyer用のSELL */
func (ai *AI) Sell(candle model.Candle, price, bbRate float64) (childOrderAcceptanceID string, isOrderCompleted bool, orderPrice float64) {
	orderPrice = 0.0
	pnl := 0.0
	size = 0.0
	atr, _ := service.Atr(30)

	if ai.StartTrade.After(candle.Time) {
		log.Println("candle.TimeがStartTradeより過去のため取引しません")
		return "timeError", false, 0.0
	}
	// ロングの利益確定後にショートでインしないようにreOpenをfalseにする
	if isLongProfit {
		shortReOpen = false
	}
	if !ai.SignalEvents.CanSell(candle.Time, shortReOpen) {
		log.Println("canSell: falseのためreturn")
		return
	}

	if !ai.BackTest {
		availableCurrency := ai.GetAvailableBalance()
		// 使用して良い金額は証拠金に3.5をかけた数とする
		useCurrency := availableCurrency * ai.UsePercent
		ticker, err := ai.API.GetTicker(ai.ProductCode)
		if err != nil {
			log.Println(err)
			return
		}
		if ticker == nil {
			return
		}
		// 証拠金の4倍でどれだけ買えるか調査
		size = 1.0 / (ticker.BestAsk / useCurrency)
		size = ai.AdjustSize(size)

		params := map[string]string{
			"product_code": "FX_BTC_JPY",
		}
		positionRes, _ := ai.API.GetPositions(params)
		fmt.Println("positionRessssssss")
		fmt.Println(positionRes)
		// (注文単位で配列で返却される)positionResが1以上の場合、注文を決済するのでSizeを格納する
		// pnl: 利益
		if len(positionRes) > 0 {
			// positionResがあった場合、sizeを初期化（上記で新規購入のsizeを出しているので決済のsizeで上書きする）
			size = 0.0
			for _, position := range positionRes {
				size += position.Size
				pnl += position.Pnl
			}
			size = math.Round(size*10000) / 10000
		}
		if math.IsNaN(size) {
			log.Println("sizeの計算が出来ませんでした。SELLを中止します。")
			return
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
			log.Printf("order=%+v status=no_id（不正なsize指定がされている可能性があります。）", order)
			return
		}
		childOrderAcceptanceID = resp.ChildOrderAcceptanceID
		isOrderCompleted, orderPrice = ai.WaitUntilOrderComplete(childOrderAcceptanceID, pnl, bbRate)
		// continueフラグがtrueのときは連続売買する。positionResが0件のときは新規なのでReOpenはしない。ADD:ロングにて利益確定済みじゃないとき（isLongProfit）
		if config.Config.Continue && len(positionRes) > 0 && !isLongProfit {
			shortReOpen = true
		}
		// StopLimit後はreOpenしない
		if (config.Config.Continue && isStopLimit) || !config.Config.Continue {
			shortReOpen = false
		}
		return childOrderAcceptanceID, isOrderCompleted, orderPrice
	} else {
		couldSell := ai.SignalEvents.Sell(ai.ProductCode, time.Now(), candle.Close, 1.0, true, shortReOpen, price, atr, pnl, bbRate)
		utils.SendLine("couldSell： " + strconv.FormatBool(couldSell))
		log.Printf("couldSell: %s", strconv.FormatBool(couldSell))
		return "", couldSell, orderPrice
	}
}

var profit float64    // オープン時に設定する1取引ごとの利益
var stopLimit float64 // 損切りライン
var atrRate float64   // atr率
var isLongProfit bool
var isShortProfit bool
var size float64
var sellOpen bool
var buyOpen bool
var isNoPosition bool // 取引中じゃない状態
//var isCandleOpportunity bool // キャンドルでの取引機会（Profit以外を指す）
var isStopLimit bool // 損切りを行った後にreOpenさせないためのフラグ

//var count int

func (ai *AI) Trade(ticker bitflyer.Ticker) {
	eventLength := model.GetAllSignalEventsCount()

	// TODO 関数にできる
	if eventLength%2 == 0 {
		isNoPosition = true
	} else {
		isNoPosition = false
	}
	if !shortReOpen && !longReOpen && time.Now().Minute()%tradeDuration != 0 && time.Now().Second() != 0 && isNoPosition {
		fmt.Printf("フラット（reOpenが無い && positionがない）状態かつ15分00秒じゃないため取引はしません。%s\n", time.Now().Truncate(time.Second))
		return
	}
	atr, _ := service.Atr(30)
	price := ticker.GetMidPrice()
	// ボラティリティが低い時はトレードしない
	fmt.Println(atr)
	if atr > 0 && eventLength%2 == 0 {
		atrRate = (float64(atr) / price) * 100
		if atrRate < 0.10 {
			log.Printf("低ボラティリティのため取引しません。（atrRate:%s\n", strconv.FormatFloat(atrRate, 'f', -1, 64))
			atrRate = 0.0
			return
		} else {
			fmt.Printf("atrRate:%s\n", strconv.FormatFloat(atrRate, 'f', -1, 64))
		}
	}
	fmt.Printf("eventLength:%s\n", strconv.Itoa(eventLength))
	// 取引が完了していたらParamsを更新する
	if eventLength%2 == 0 {
		reOpen := false
		if longReOpen || shortReOpen {
			reOpen = true
		}
		if ai.OptimizedTradeParams == nil {
			go ai.UpdateOptimizeParams(true, reOpen)
		}
	}
	// goroutineの同時実行数を制御
	isAcquire := ai.TradeSemaphore.TryAcquire(1)
	if !isAcquire {
		log.Println("Could not get trade lock")
		return
	}
	defer ai.TradeSemaphore.Release(1)
	params := ai.OptimizedTradeParams
	log.Println(params)
	log.Printf("profit:%s\n", strconv.FormatFloat(profit, 'f', -1, 64))
	if params == nil {
		return
	}
	df, _ := service.GetAllCandle(ai.ProductCode, ai.Duration, ai.PastPeriod)
	lenCandles := len(df.Candles)
	params.EmaEnable = true
	params.MacdEnable = true

	// EMA
	var emaValues1 []float64
	var emaValues2 []float64
	// var emaValues3 []float64
	if params.EmaEnable {
		emaValues1 = talib.Ema(df.Closes(), 7)
		emaValues2 = talib.Ema(df.Closes(), 14)
		// emaValues3 = talib.Ema(df.Closes(), 50)
	}

	// ボリンジャーバンド
	var bbUp []float64
	var bbDown []float64
	params.BbEnable = true
	if params.BbEnable {
		bbUp, _, bbDown = talib.BBands(df.Closes(), 20, 2, 2, 0)
	}

	//// 一目均衡表
	//var tenkan, kijun, senkouA, senkouB, chikou []float64
	//if params.IchimokuEnable {
	//	tenkan, kijun, senkouA, senkouB, chikou = tradingalgo.IchimokuCloud(df.Closes())
	//}
	//
	// MACD
	var outMACD, outMACDSignal, outMACDHist []float64
	if params.MacdEnable {
		outMACD, outMACDSignal, outMACDHist = talib.Macd(df.Closes(), params.MacdFastPeriod, params.MacdSlowPeriod, params.MacdSignalPeriod)
	}
	//
	//// RSI
	//var rsiValues []float64
	//if params.RsiEnable {
	//	rsiValues = talib.Rsi(df.Closes(), params.RsiPeriod)
	//}
	fmt.Printf("lenCandles:%s\n", strconv.Itoa(lenCandles))
	for i := lenCandles - 2; i < lenCandles; i++ {
		// 有効なインディケータの数
		buyPoint, sellPoint := 0, 0
		// ゴールデンクロス・デッドクロスが計算できる条件
		if params.EmaEnable && params.EmaPeriod1 <= i && params.EmaPeriod2 <= i {
			// ゴールデンクロス with MACD
			// buyOpenのオープン
			//log.Printf("MACDのロング条件??: %s\n", strconv.FormatBool((outMACD[i] > 0 || outMACDHist[i] > 0) && outMACD[i] >= outMACDSignal[i]))
			//log.Printf("MACDのショート条件??: %s\n", strconv.FormatBool((outMACD[i] < 0 || outMACDHist[i] < 0) && outMACD[i] <= outMACDSignal[i]))
			// ADD: #63 MACDのメインライン（outMACD[i]）が0より大きい && シグナル（outMACDSignal）より大きい を条件として追加
			// #64 if !buyOpen && !sellOpen && emaValues1[i-1] < emaValues2[i-1] && emaValues1[i] >= emaValues2[i] && (outMACD[i] > 0 || outMACDHist[i] > 0) && outMACD[i] >= outMACDSignal[i] {
			//	buyPoint++
			//}
			// buyOpenのオープン
			if !buyOpen && !sellOpen && emaValues1[i-1] < emaValues2[i-1] && emaValues1[i] >= emaValues2[i] && (outMACD[i] > 0 || outMACDHist[i] > 0) && outMACD[i] >= outMACDSignal[i] {
				buyPoint++
			}
			// buyOpenのクローズ
			if buyOpen && !sellOpen && emaValues1[i-1] > emaValues2[i-1] && emaValues1[i] <= emaValues2[i] && (outMACD[i] < 0 || outMACDHist[i] < 0) && outMACD[i] <= outMACDSignal[i] {
				sellPoint++
			}
			// デッドクロス
			// sellOpenのオープン
			// ADD: #63 MACDのメインライン（outMACD[i]）が0より小さい && シグナル（outMACDSignal）より小さい を条件として追加
			// #64 if !buyOpen && !sellOpen && emaValues1[i-1] > emaValues2[i-1] && emaValues1[i] <= emaValues2[i] && (outMACD[i] < 0 || outMACDHist[i] < 0) && outMACD[i] <= outMACDSignal[i] {
			//	sellPoint++
			//}
			// ショートのオープン
			if !buyOpen && !sellOpen && emaValues1[i-1] > emaValues2[i-1] && emaValues1[i] <= emaValues2[i] && (outMACD[i] < 0 || outMACDHist[i] < 0) && outMACD[i] <= outMACDSignal[i] { // && pauseDone
				sellPoint++
			}
			// sellOpenのクローズ ADD: #63 MACDのメインライン（outMACD[i]）が0より大きい && シグナル（outMACDSignal）より大きい を条件として追加
			if sellOpen && !buyOpen && emaValues1[i-1] < emaValues2[i-1] && emaValues1[i] >= emaValues2[i] && (outMACD[i] > 0 || outMACDHist[i] > 0) && outMACD[i] >= outMACDSignal[i] {
				buyPoint++
			}
		}

		// ボリンジャーバンド
		//if params.BbEnable && params.BbN <= i {
		//	// 上抜け（買い）
		//	if bbDown[i-1] > df.Candles[i-1].Close && bbDown[i] <= df.Candles[i].Close {
		//		sellPoint++
		//	}
		//	// 下抜け（売り）
		//	if bbUp[i-1] < df.Candles[i-1].Close && bbUp[i] >= df.Candles[i].Close {
		//		sellPoint++
		//	}
		//}

		//// MACD
		//if params.MacdEnable {
		//	// 上抜け（買い）
		//	if outMACD[i] < 0 && outMACDSignal[i] < 0 && outMACD[i-1] < outMACDSignal[i-1] && outMACD[i] >= outMACDSignal[i] {
		//		buyPoint++
		//	}
		//	// 下抜け（売り）
		//	if outMACD[i] > 0 && outMACDSignal[i] > 0 && outMACD[i-1] > outMACDSignal[i-1] && outMACD[i] <= outMACDSignal[i] {
		//		sellPoint++
		//	}
		//}
		//// 一目均衡表
		//if params.IchimokuEnable {
		//	if chikou[i-1] < df.Candles[i-1].High && chikou[i] >= df.Candles[i].High &&
		//		senkouA[i] < df.Candles[i].Low && senkouB[i] < df.Candles[i].Low &&
		//		tenkan[i] > kijun[i] {
		//		buyPoint++
		//	}
		//
		//	if chikou[i-1] > df.Candles[i-1].Low && chikou[i] <= df.Candles[i].Low &&
		//		senkouA[i] > df.Candles[i].High && senkouB[i] > df.Candles[i].High &&
		//		tenkan[i] < kijun[i] {
		//		sellPoint++
		//	}
		//}
		//// RSI
		//if params.RsiEnable && rsiValues[i-1] != 0 && rsiValues[i-1] != 100 {
		//	// 30% 上抜け（買い）
		//	if rsiValues[i-1] < params.RsiBuyThread && rsiValues[i] >= params.RsiBuyThread {
		//		buyPoint++
		//	}
		//	// 70% 下抜け（売り）
		//	if rsiValues[i-1] > params.RsiSellThread && rsiValues[i] <= params.RsiSellThread {
		//		sellPoint++
		//	}
		//}
		// オープンの場合はbuyPoint,sellPointどちらかが2以上のときでStopLimitを設定する
		bbRate := 1.0
		bbWith := 0.0
		if bbUp == nil || bbDown == nil {
			log.Println("bbUpまたはbbDownがnilのため取引をしません")
			return
		}
		if len(bbUp) >= i && len(bbDown) >= i {
			bbRate = bbDown[i] / bbUp[i]
		}
		if len(bbUp) >= i && len(bbDown) >= i {
			bbWith = (bbUp[i] / bbDown[i]) - 1.0
		}
		log.Printf("オープン可能かどうか：%s\n", strconv.FormatBool(isNoPosition && bbWith > config.Config.OpenableBbWith && bbRate < config.Config.OpenableBbRate || (shortReOpen || longReOpen)))
		log.Println("--------------------------以下、詳細です--------------------------")
		log.Printf("bbRate:%s\n", strconv.FormatFloat(bbRate, 'f', -1, 64))
		log.Printf("bbWith:%s\n", strconv.FormatFloat(bbWith, 'f', -1, 64))
		log.Printf("isNoPosition:%s\n", strconv.FormatBool(isNoPosition))
		log.Printf("sellOpen?:%s\n", strconv.FormatBool(sellOpen))
		log.Printf("buyOpen?:%s\n", strconv.FormatBool(buyOpen))
		if time.Now().Minute() == 0 || (shortReOpen || longReOpen) {
			if isNoPosition && bbWith > config.Config.OpenableBbWith && bbRate < config.Config.OpenableBbRate || (shortReOpen || longReOpen) {
				// 1つでも買いのインディケータがあれば買い
				// #64 if sellPoint > buyPoint || (shortReOpen && (outMACD[i] < 0 || outMACDHist[i] < 0) && outMACD[i] <= outMACDSignal[i]) {
				log.Printf("ショート？？:%s\n", strconv.FormatBool(sellPoint > buyPoint))
				if sellPoint > buyPoint || shortReOpen {
					childOrderAcceptanceID, isOrderCompleted, orderPrice := ai.Sell(df.Candles[i], price, bbRate)
					log.Printf("childOrderAcceptanceID: %s", childOrderAcceptanceID)
					if childOrderAcceptanceID == "timeError" {
						continue
					}
					if !isOrderCompleted {
						utils.SendLine("オープンショート：注文が保存できませんでした。logを確認してください。")
						log.Println("オープンショート：注文が保存できませんでした。logを確認してください。")
						continue
					}
					// StopLimit後のオープンの場合はisStopLimitを初期化する
					if isStopLimit {
						isStopLimit = false
					}
					// ロングの利確後のオープンの場合はisLongProfitを初期化する
					if isLongProfit {
						isLongProfit = false
					}
					log.Printf("bbRate:%s\n", strconv.FormatFloat(bbRate, 'f', -1, 64))
					if ai.BackTest {
						orderPrice = price
					}
					//profit = math.Floor(orderPrice*0.996*10000) / 10000
					// オープン時にボリンジャーバンドの下抜け値をターゲットに設定
					if len(bbDown) >= i {
						//profit = bbDown[i] * 0.997
						profit = bbDown[i] * 0.99
						//profit = orderPrice * 0.9997
						log.Printf("profit(bbDownから):%s\n", strconv.FormatFloat(profit, 'f', -1, 64))
					} else {
						profit = math.Floor(orderPrice*0.975*10000) / 10000
						//profit = math.Floor(orderPrice*0.9995*10000) / 10000
						log.Printf("profit(bbDownから取れなかったのでパーセントで出す):%s\n", strconv.FormatFloat(profit, 'f', -1, 64))
					}
					// ボリンジャーバンドの下抜け値がorderPriceより小さかったらorderPriceから利益を算出する
					if len(bbDown) >= i {
						if orderPrice < bbDown[i] {
							log.Println("急激な値の変化です。bbandsは使わずに%で利益を決定します。")
							profit = math.Floor(orderPrice*0.975*10000) / 10000
							//profit = math.Floor(orderPrice*0.9995*10000) / 10000
							log.Println(profit)
						}
					}
					stopLimit = orderPrice * (1.0 + (1.0 - ai.StopLimitPercent))
					log.Printf("orderPrice:%s\n", strconv.FormatFloat(orderPrice, 'f', -1, 64))
					log.Printf("profit:%s\n", strconv.FormatFloat(profit, 'f', -1, 64))
					log.Println("sellOpenのオープン")
					utils.SendLine("ショートのオープン（sell): " + strconv.FormatFloat(orderPrice, 'f', -1, 64) + "\nstopLimit: " + strconv.FormatFloat(stopLimit, 'f', -1, 64) + "\nbbRate: " + strconv.FormatFloat(bbRate, 'f', -1, 64) + "\nbbWith: " + strconv.FormatFloat(bbWith, 'f', -1, 64))
					sellOpen = true
					if shortReOpen {
						log.Println("shortReOpen成功")
						shortReOpen = false
					}
				}
				// #64
				//if buyPoint > sellPoint || (longReOpen && (outMACD[i] > 0 || outMACDHist[i] > 0) && outMACD[i] >= outMACDSignal[i]) {
				log.Printf("ロング？？buyPoint > sellPoint:%s\n", strconv.FormatBool(buyPoint > sellPoint))
				if buyPoint > sellPoint || longReOpen {
					childOrderAcceptanceID, isOrderCompleted, orderPrice := ai.Buy(df.Candles[i], price, bbRate)
					if childOrderAcceptanceID == "timeError" {
						continue
					}
					if !isOrderCompleted {
						utils.SendLine("オープンロング：注文が保存できませんでした。logを確認してください。")
						log.Println("オープンロング：注文が保存できませんでした。logを確認してください。")
						continue
					}
					// StopLimit後のオープンの場合はisStopLimitを初期化する
					if isStopLimit {
						isStopLimit = false
					}
					// ショートの利確後のオープンの場合はisShortProfitを初期化する
					if isShortProfit {
						isShortProfit = false
					}
					log.Printf("bbRate:%s\n", strconv.FormatFloat(bbRate, 'f', -1, 64))
					if ai.BackTest {
						orderPrice = price
					}
					//profit = math.Floor(orderPrice*1.004*10000) / 10000
					// オープン時にボリンジャーバンドの上抜けけ値をターゲットに設定
					if len(bbUp) >= i {
						//profit = bbUp[i] * 1.003
						profit = bbUp[i] * 1.01
						// profit = bbUp[i]
						//profit = orderPrice * 1.0003
						log.Printf("profit(bbUpから):%s\n", strconv.FormatFloat(profit, 'f', -1, 64))
					} else {
						profit = math.Floor(orderPrice*1.025*10000) / 10000
						//profit = math.Floor(orderPrice*1.0005*10000) / 10000
						log.Printf("profit(bbUpから取れなかったのでパーセントで):%s\n", strconv.FormatFloat(profit, 'f', -1, 64))
					}
					if len(bbUp) >= i {
						if orderPrice > bbUp[i] {
							log.Println("急激な値の変化です。bbandsは使わずに%で利益を決定します。")
							profit = math.Floor(orderPrice*1.025*10000) / 10000
							//profit = math.Floor(orderPrice* 1.0005*10000) / 10000
							log.Println(profit)
						}
					}
					stopLimit = orderPrice * ai.StopLimitPercent
					log.Printf("orderPrice:%s\n", strconv.FormatFloat(orderPrice, 'f', -1, 64))
					log.Printf("profit:%s\n", strconv.FormatFloat(profit, 'f', -1, 64))
					log.Println("buyOpenのオープン")
					utils.SendLine("ロングのオープン（buy): " + strconv.FormatFloat(orderPrice, 'f', -1, 64) + "\nstopLimit: " + strconv.FormatFloat(stopLimit, 'f', -1, 64) + "\nbbRate: " + strconv.FormatFloat(bbRate, 'f', -1, 64) + "\nbbWith: " + strconv.FormatFloat(bbWith, 'f', -1, 64))
					buyOpen = true
					if longReOpen {
						log.Println("longReOpen成功")
						longReOpen = false
					}
				}
			}
		}
		// クローズ
		// クローズ時はbuyPoint, sellPointどちらも1以上でParamsをUpdateしてStopLimitを初期化
		// sellOpenのクローズ（buyPointにてクローズする場合は15分単位のみ）
		//if sellOpen == true && (buyPoint > 0 || price <= profit || price >= stopLimit) {
		if sellOpen {
			log.Printf("クローズsellOpen?:%s\n", strconv.FormatBool(sellOpen))
			log.Printf("クローズショート？？buyPoint > sellPoint:%s\n", strconv.FormatBool(buyPoint > sellPoint))
			log.Printf("クローズショート？？price <= profit:%s\n", strconv.FormatBool(price <= profit))
			log.Printf("クローズショート？？総合判定:%s\n", strconv.FormatBool((buyPoint > 0 && time.Now().Minute()%tradeDuration == 0 && time.Now().Second() < 5) || (price <= profit || price >= stopLimit)))
			if buyPoint > 0 || price <= profit || price >= stopLimit {
				_, isOrderCompleted, _ := ai.Buy(df.Candles[i], price, bbRate)
				if !isOrderCompleted {
					utils.SendLine("クローズショート：注文が保存できませんでした。logを確認してください。")
					log.Println("クローズショート：注文が保存できませんでした。logを確認してください。")
					continue
				}
				if price <= profit {
					isShortProfit = true
				}
				if price >= stopLimit {
					log.Println("損切り")
					isStopLimit = true
				}
				utils.SendLine("ショートのクローズ（buy): " + strconv.FormatFloat(price, 'f', -1, 64))
				fmt.Printf("priceの値:%s\n", strconv.FormatFloat(price, 'f', -1, 64))
				fmt.Printf("isProfit??: %s\n", strconv.FormatBool(price <= profit))
				fmt.Printf("Profitの値:%s\n", strconv.FormatFloat(profit, 'f', -1, 64))
				fmt.Printf("isStopLimit??: %s\n", strconv.FormatBool(price >= stopLimit))
				fmt.Printf("StopLimitの値:%s\n", strconv.FormatFloat(stopLimit, 'f', -1, 64))
				log.Println("sellOpenのクローズ")
				sellOpen = false
				profit = 0.0
				stopLimit = 0.0
				// ai.UpdateOptimizeParams(true)
			}
		}
		// buyOpenのクローズ（sellPointにてクローズする場合は15分単位のみ）
		if buyOpen {
			log.Printf("クローズbuyOpen?:%s\n", strconv.FormatBool(buyOpen))
			log.Printf("クローズロングbuyPoint > sellPoint:%s\n", strconv.FormatBool(buyPoint < sellPoint))
			log.Printf("クローズロングprice >= profit:%s\n", strconv.FormatBool(price >= profit))
			log.Printf("クローズロング最終判定:%s\n", strconv.FormatBool((sellPoint > 0 && time.Now().Minute()%tradeDuration == 0 && time.Now().Second() < 5) || (price >= profit || price <= stopLimit)))
			if sellPoint > 0 || price >= profit || price <= stopLimit {
				_, isOrderCompleted, _ := ai.Sell(df.Candles[i], price, bbRate)
				if !isOrderCompleted {
					utils.SendLine("クローズロング：注文が保存できませんでした。logを確認してください。")
					log.Println("クローズロング：注文が保存できませんでした。logを確認してください。")
					continue
				}
				if price >= profit {
					isLongProfit = true
				}
				if price <= stopLimit {
					log.Println("損切り")
					isStopLimit = true
				}
				utils.SendLine("ロングのクローズ（sell): " + strconv.FormatFloat(price, 'f', -1, 64))
				log.Println("buyOpenのクローズ")
				fmt.Printf("priceの値:%s\n", strconv.FormatFloat(price, 'f', -1, 64))
				fmt.Printf("isProfit??: %s\n", strconv.FormatBool(price <= profit))
				fmt.Printf("Profitの値:%s\n", strconv.FormatFloat(profit, 'f', -1, 64))
				fmt.Printf("isStopLimit??: %s\n", strconv.FormatBool(price >= stopLimit))
				fmt.Printf("StopLimitの値:%s\n", strconv.FormatFloat(stopLimit, 'f', -1, 64))
				buyOpen = false
				profit = 0.0
				stopLimit = 0.0
				// ai.UpdateOptimizeParams(true)
			}
		}
		// 1つでも買いのインディケータがあれば買い
		//if buyPoint > 0 {
		//	_, isOrderCompleted := ai.Sell(df.Candles[i])
		//	if !isOrderCompleted {
		//		continue
		//	}
		//	buyOpen = false
		//	stopLimit = 0.0
		//	ai.UpdateOptimizeParams(true)
		//}
		//if sellPoint > 0 {
		//	_, isOrderCompleted := ai.Buy(df.Candles[i])
		//	if !isOrderCompleted {
		//		continue
		//	}
		//	sellOpen = false
		//	stopLimit = 0.0
		//	ai.UpdateOptimizeParams(true)
		//}
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
func (ai *AI) WaitUntilOrderComplete(childOrderAcceptanceID string, pnl, bbRate float64) (bool, float64) {
	atr, _ := service.Atr(30)
	params := map[string]string{
		"product_code":              ai.ProductCode,
		"child_order_acceptance_id": childOrderAcceptanceID,
	}
	expire := time.After(time.Second * 30)
	interval := time.Tick(5 * time.Second)
	return func() (bool, float64) {
		for {
			select {
			case <-expire:
				return false, 0
			case <-interval:
				listOrders, err := ai.API.ListOrder(params)
				if err != nil {
					log.Println(err)
					return false, 0
				}
				if len(listOrders) == 0 {
					return false, 0
				}
				order := listOrders[0]
				if order.ChildOrderState == "COMPLETED" {
					if order.Side == "BUY" {
						couldBuy := ai.SignalEvents.Buy(ai.ProductCode, time.Now().Truncate(time.Second), order.AveragePrice, order.Size, true, longReOpen, order.AveragePrice, atr, pnl, bbRate)
						if !couldBuy {
							log.Printf("status=buy childOrderAcceptanceID=%s order=%+v", childOrderAcceptanceID, order)
						}
						return couldBuy, order.AveragePrice
					}
					if order.Side == "SELL" {
						couldSell := ai.SignalEvents.Sell(ai.ProductCode, time.Now().Truncate(time.Second), order.AveragePrice, order.Size, true, shortReOpen, order.AveragePrice, atr, pnl, bbRate)
						if !couldSell {
							log.Printf("status=sell childOrderAcceptanceID=%s order=%+v", childOrderAcceptanceID, order)
						}
						return couldSell, order.AveragePrice
					}
					return false, 0
				}
			}
		}
	}()
}
