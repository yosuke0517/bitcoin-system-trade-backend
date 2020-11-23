package model

import (
	"app/config"
	"app/domain/tradingalgo"
	"github.com/markcheno/go-talib"
	"os"
	"sort"
	"strconv"
	"time"
)

type DataFrameCandle struct {
	ProductCode   string         `json:"product_code"`
	Duration      time.Duration  `json:"duration"`
	Candles       []Candle       `json:"candles"`
	Smas          []Sma          `json:"smas,omitempty"`
	Emas          []Ema          `json:"emas,omitempty"`
	BBands        *BBands        `json:"bbands,omitempty"` // スライスじゃない場合はポインタ（ポインタがないとStructの空のjsonを返してしまいomitemptyが効かない）
	IchimokuCloud *IchimokuCloud `json:"ichimoku,omitempty"`
	Rsi           *Rsi           `json:"rsi,omitempty"`
	Macd          *Macd          `json:"macd,omitempty"`
	Hvs           []Hv           `json:"hvs,omitempty"`
	Events        *SignalEvents  `json:"events,omitempty"`
}

/** 単純移動平均線 */
type Sma struct {
	Period int       `json:"period,omitempty"` // omitempty←データがない時はjsonとして返却しない
	Values []float64 `json:"values,omitempty"`
}

/** 指数平滑移動平均線 */
type Ema struct {
	Period int       `json:"period,omitempty"` // omitempty←データがない時はjsonとして返却しない
	Values []float64 `json:"values,omitempty"`
}

/** ボリンジャーバンド */
type BBands struct {
	N    int       `json:"n,omitempty"`    // 移動平均線期間
	K    float64   `json:"k,omitempty"`    // 標準偏差２個（上下の線を算出するために必要）
	Up   []float64 `json:"up,omitempty"`   // 上のライン
	Mid  []float64 `json:"mid,omitempty"`  // 中央のライン
	Down []float64 `json:"down,omitempty"` // 下のライン
}

/** 一目均衡表 */
type IchimokuCloud struct {
	Tenkan  []float64 `json:"tenkan,omitempty"`
	Kijun   []float64 `json:"kijun,omitempty"`
	SenkouA []float64 `json:"senkoua,omitempty"`
	SenkouB []float64 `json:"senkoub,omitempty"`
	Chikou  []float64 `json:"chikou,omitempty"`
}

/** RSI（買われすぎ, 売られすぎの指標）*/
type Rsi struct {
	Period int       `json:"period,omitempty"`
	Values []float64 `json:"values,omitempty"`
}

/** MACD */
type Macd struct {
	FastPeriod   int       `json:"fast_period,omitempty"`
	SlowPeriod   int       `json:"slow_period,omitempty"`
	SignalPeriod int       `json:"signal_period,omitempty"`
	Macd         []float64 `json:"macd,omitempty"`
	MacdSignal   []float64 `json:"macd_signal,omitempty"`
	MacdHist     []float64 `json:"macd_hist,omitempty"`
}

/** ヒストリカルボラティリティ （値動きの激しさを計る指標）*/
type Hv struct {
	Period int       `json:"period,omitempty"`
	Values []float64 `json:"values,omitempty"`
}

/** Timeのみをスライスで返す */
func (df *DataFrameCandle) Times() []time.Time {
	s := make([]time.Time, len(df.Candles))
	for i, candle := range df.Candles {
		s[i] = candle.Time
	}
	return s
}

/** Openのみをスライスで返す */
func (df *DataFrameCandle) Opens() []float64 {
	s := make([]float64, len(df.Candles))
	for i, candle := range df.Candles {
		s[i] = candle.Open
	}
	return s
}

/** Closeのみをスライスで返す */
func (df *DataFrameCandle) Closes() []float64 {
	s := make([]float64, len(df.Candles))
	for i, candle := range df.Candles {
		s[i] = candle.Close
	}
	return s
}

/** Highのみをスライスで返す */
func (df *DataFrameCandle) Highs() []float64 {
	s := make([]float64, len(df.Candles))
	for i, candle := range df.Candles {
		s[i] = candle.High
	}
	return s
}

/** Lowのみをスライスで返す */
func (df *DataFrameCandle) Low() []float64 {
	s := make([]float64, len(df.Candles))
	for i, candle := range df.Candles {
		s[i] = candle.Low
	}
	return s
}

/** Lowのみをスライスで返す */
func (df *DataFrameCandle) Volume() []float64 {
	s := make([]float64, len(df.Candles))
	for i, candle := range df.Candles {
		s[i] = candle.Volume
	}
	return s
}

/** 単純移動平均線 */
func (df *DataFrameCandle) AddSma(period int) bool {
	// ex) period = 14
	if len(df.Candles) > period {
		df.Smas = append(df.Smas, Sma{
			Period: period,
			Values: talib.Sma(df.Closes(), period),
		})
		return true
	}
	return false
}

/** 指数平滑移動平均線 */
func (df *DataFrameCandle) AddEma(period int) bool {
	// ex) period = 14
	if len(df.Candles) > period {
		df.Emas = append(df.Emas, Ema{
			Period: period,
			Values: talib.Ema(df.Closes(), period),
		})
		return true
	}
	return false
}

/** ボリンジャーバンド
n: 移動平均線の期間（デフォルト20）
k: 標準偏差２個分（デフォルト2）
*/
func (df *DataFrameCandle) AddBBands(n int, k float64) bool {
	if n <= len(df.Closes()) {
		up, mid, down := talib.BBands(df.Closes(), n, k, k, 0)
		df.BBands = &BBands{
			N:    n,
			K:    k,
			Up:   up,
			Mid:  mid,
			Down: down,
		}
		return true
	}
	return false
}

/** 一目均衡表 */
func (df *DataFrameCandle) AddIchimoku() bool {
	tenkanN := 9
	if len(df.Closes()) >= tenkanN {
		tenkan, kijun, senkouA, senkouB, chikou := tradingalgo.IchimokuCloud(df.Closes())
		df.IchimokuCloud = &IchimokuCloud{
			Tenkan:  tenkan,
			Kijun:   kijun,
			SenkouA: senkouA,
			SenkouB: senkouB,
			Chikou:  chikou,
		}
		return true
	}
	return false
}

/** RSI */
func (df *DataFrameCandle) AddRsi(period int) bool {
	if len(df.Candles) > period {
		values := talib.Rsi(df.Closes(), period)
		df.Rsi = &Rsi{
			Period: period,
			Values: values,
		}
		return true
	}
	return false
}

/** MACD */
func (df *DataFrameCandle) AddMacd(inFastPeriod, inSlowPeriod, inSignalPeriod int) bool {
	if len(df.Candles) > 1 {
		outMACD, outMACDSignal, outMACDHist := talib.Macd(df.Closes(), inFastPeriod, inSlowPeriod, inSignalPeriod)
		df.Macd = &Macd{
			FastPeriod:   inFastPeriod,
			SlowPeriod:   inSlowPeriod,
			SignalPeriod: inSignalPeriod,
			Macd:         outMACD,
			MacdSignal:   outMACDSignal,
			MacdHist:     outMACDHist,
		}
		return true
	}
	return false
}

/** ヒストリカルボラティリティ */
func (df *DataFrameCandle) AddHv(period int) bool {
	if len(df.Candles) >= period {
		df.Hvs = append(df.Hvs, Hv{
			Period: period,
			Values: tradingalgo.Hv(df.Closes(), period),
		})
		return true
	}
	return false
}

func (df *DataFrameCandle) AddEvents(timeTime time.Time) bool {
	signalEvents := GetSignalEventsAfterTime(timeTime)
	if len(signalEvents.Signals) > 0 {
		df.Events = signalEvents
		return true
	}
	return false
}

/** EMAバックテスト */
func (df *DataFrameCandle) BackTestEma(period1, period2 int) *SignalEvents {
	lenCandles := len(df.Candles)
	// キャンドル数が足りているかのチェック
	if lenCandles <= period1 || lenCandles <= period2 {
		return nil
	}
	signalEvents := NewSignalEvents()
	// EMAを算出
	emaValue1 := talib.Ema(df.Closes(), period1)
	emaValue2 := talib.Ema(df.Closes(), period2)
	// 指定の数までは値が0で入ってくるので飛ばす （7の場合の例：0,0,0,0,0,0,1005000,）
	for i := 1; i < lenCandles; i++ {
		if i < period1 || i < period2 {
			continue
		}
		// ゴールデンクロス時は買い
		if emaValue1[i-1] < emaValue2[i-1] && emaValue1[i] >= emaValue2[i] {
			signalEvents.Buy(df.ProductCode, df.Candles[i].Time, df.Candles[i].Close, 1.0, false)
		}
		// デッドクロスは売り
		if emaValue1[i-1] > emaValue2[i-1] && emaValue1[i] <= emaValue2[i] {
			signalEvents.Sell(df.ProductCode, df.Candles[i].Time, df.Candles[i].Close, 1.0, false)
		}
	}
	return signalEvents
}

/** EMA最適化
利益が出ないと判断すれば0, 7, 14を返す
*/
func (df *DataFrameCandle) OptimizeEma() (performance float64, bestPeriod1 int, bestPeriod2 int) {
	bestPeriod1 = 7
	bestPeriod2 = 14
	// TODO 数を伸ばしたりして要調整 No.129
	for period1 := 5; period1 < 50; period1++ {
		for period2 := 12; period2 < 50; period2++ {
			signalEvents := df.BackTestEma(period1, period2)
			if signalEvents == nil {
				continue
			}
			// それぞれの利益を出して1番良い成績を残す日数を探す
			profit := signalEvents.Profit()
			if performance < profit {
				performance = profit
				bestPeriod1 = period1
				bestPeriod2 = period2
			}
		}
	}
	return performance, bestPeriod1, bestPeriod2
}

/** ボリンジャーバンドバックテスト */
func (df *DataFrameCandle) BackTestBb(n int, k float64) *SignalEvents {
	lenCandles := len(df.Candles)
	// キャンドル数チェック
	if lenCandles <= n {
		return nil
	}

	signalEvents := &SignalEvents{}
	bbUp, _, bbDown := talib.BBands(df.Closes(), n, k, k, 0)
	// i < nの時は０が返ってくる？のでスキップ
	for i := 1; i < lenCandles; i++ {
		if i < n {
			continue
		}
		// 買い（売られ過ぎ）判定
		if bbDown[i-1] > df.Candles[i-1].Close && bbDown[i] <= df.Candles[i].Close {
			signalEvents.Buy(df.ProductCode, df.Candles[i].Time, df.Candles[i].Close, 1.0, false)
		}
		// 売り（買われ過ぎ）判定
		if bbUp[i-1] < df.Candles[i-1].Close && bbUp[i] >= df.Candles[i].Close {
			signalEvents.Sell(df.ProductCode, df.Candles[i].Time, df.Candles[i].Close, 1.0, false)
		}
	}
	return signalEvents
}

/** ボリンジャーバンド最適化 */
func (df *DataFrameCandle) OptimizeBb() (performance float64, bestN int, bestK float64) {
	bestN = 20
	bestK = 2.0
	// TDOO 数を増やせば範囲が広がる（例 n := 10 n < 60 TODO No.130
	for n := 10; n < 20; n++ {
		// 1.0 , 3.0とかにすると範囲が広がり緩くなる
		for k := 1.9; k < 2.1; k += 0.1 {
			signalEvents := df.BackTestBb(n, k)
			if signalEvents == nil {
				continue
			}
			profit := signalEvents.Profit()
			if performance < profit {
				performance = profit
				bestN = n
				bestK = k
			}
		}
	}
	return performance, bestN, bestK
}

/** 一目均衡表 */
func (df *DataFrameCandle) BackTestIchimoku() *SignalEvents {
	lenCandles := len(df.Candles)

	if lenCandles <= 52 {
		return nil
	}

	var signalEvents SignalEvents
	tenkan, kijun, senkouA, senkouB, chikou := tradingalgo.IchimokuCloud(df.Closes())
	for i := 1; i < lenCandles; i++ {
		// 買い判定（三役好転）
		if chikou[i-1] < df.Candles[i-1].High && chikou[i] >= df.Candles[i].High &&
			senkouA[i] < df.Candles[i].Low && senkouB[i] < df.Candles[i].Low &&
			tenkan[i] > kijun[i] {
			signalEvents.Buy(df.ProductCode, df.Candles[i].Time, df.Candles[i].Close, 1.0, false)
		}
		// 売り判定（三役逆転）
		if chikou[i-1] > df.Candles[i-1].Low && chikou[i] <= df.Candles[i].Low &&
			senkouA[i] > df.Candles[i].High && senkouB[i] > df.Candles[i].High &&
			tenkan[i] < kijun[i] {
			signalEvents.Sell(df.ProductCode, df.Candles[i].Time, df.Candles[i].Close, 1.0, false)
		}
	}
	return &signalEvents
}

// 一目均衡表最適化
func (df *DataFrameCandle) OptimizeIchimoku() (performance float64) {
	signalEvents := df.BackTestIchimoku()
	if signalEvents == nil {
		return 0.0
	}
	performance = signalEvents.Profit()
	return performance
}

/** MACDバックテスト */
func (df *DataFrameCandle) BackTestMacd(macdFastPeriod, macdSlowPeriod, macdSignalPeriod int) *SignalEvents {
	lenCandles := len(df.Candles)

	if lenCandles <= macdFastPeriod || lenCandles <= macdSlowPeriod || lenCandles <= macdSignalPeriod {
		return nil
	}

	signalEvents := &SignalEvents{}
	outMACD, outMACDSignal, _ := talib.Macd(df.Closes(), macdFastPeriod, macdSlowPeriod, macdSignalPeriod)
	for i := 1; i < lenCandles; i++ {
		// 買い判定
		if outMACD[i] < 0 &&
			outMACDSignal[i] < 0 &&
			outMACD[i-1] < outMACDSignal[i-1] &&
			outMACD[i] >= outMACDSignal[i] {
			signalEvents.Buy(df.ProductCode, df.Candles[i].Time, df.Candles[i].Close, 1.0, false)
		}
		// 売り判定
		if outMACD[i] > 0 &&
			outMACDSignal[i] > 0 &&
			outMACD[i-1] > outMACDSignal[i-1] &&
			outMACD[i] <= outMACDSignal[i] {
			signalEvents.Sell(df.ProductCode, df.Candles[i].Time, df.Candles[i].Close, 1.0, false)
		}
	}
	return signalEvents
}

/** MACD最適化 */
func (df *DataFrameCandle) OptimizeMacd() (performance float64, bestMacdFastPeriod, bestMacdSlowPeriod, bestMacdSignalPeriod int) {
	bestMacdFastPeriod = 12
	bestMacdSlowPeriod = 26
	bestMacdSignalPeriod = 9

	for fastPeriod := 10; fastPeriod < 25; fastPeriod++ {
		for slowPeriod := 20; slowPeriod < 33; slowPeriod++ {
			for signalPeriod := 5; signalPeriod < 18; signalPeriod++ {
				signalEvents := df.BackTestMacd(bestMacdFastPeriod, bestMacdSlowPeriod, bestMacdSignalPeriod)
				if signalEvents == nil {
					continue
				}
				profit := signalEvents.Profit()
				if performance < profit {
					performance = profit
					bestMacdFastPeriod = fastPeriod
					bestMacdSlowPeriod = slowPeriod
					bestMacdSignalPeriod = signalPeriod
				}
			}
		}
	}
	return performance, bestMacdFastPeriod, bestMacdSlowPeriod, bestMacdSignalPeriod
}

/** RSIバックテスト */
func (df *DataFrameCandle) BackTestRsi(period int, buyThread, sellThread float64) *SignalEvents {
	lenCandles := len(df.Candles)
	if lenCandles <= period {
		return nil
	}

	signalEvents := NewSignalEvents()
	values := talib.Rsi(df.Closes(), period)
	for i := 1; i < lenCandles; i++ {
		if values[i-1] == 0 || values[i-1] == 100 {
			continue
		}
		if values[i-1] < buyThread && values[i] >= buyThread {
			signalEvents.Buy(df.ProductCode, df.Candles[i].Time, df.Candles[i].Close, 1.0, false)
		}

		if values[i-1] > sellThread && values[i] <= sellThread {
			signalEvents.Sell(df.ProductCode, df.Candles[i].Time, df.Candles[i].Close, 1.0, false)
		}
	}
	return signalEvents
}

/** RSI最適化
bestBuyThread: 買いのライン, bestSellThread: 売りのライン
*/
func (df *DataFrameCandle) OptimizeRsi() (performance float64, bestPeriod int, bestBuyThread, bestSellThread float64) {
	// 各デフォルト
	bestPeriod = 14
	bestBuyThread, bestSellThread = 30.0, 70.0

	for period := 5; period < 30; period++ {
		signalEvents := df.BackTestRsi(period, bestBuyThread, bestSellThread)
		if signalEvents == nil {
			continue
		}
		profit := signalEvents.Profit()
		if performance < profit {
			performance = profit
			bestPeriod = period
			bestBuyThread = bestBuyThread
			bestSellThread = bestSellThread
		}
	}
	return performance, bestPeriod, bestBuyThread, bestSellThread
}

type TradeParams struct {
	EmaEnable        bool
	EmaPeriod1       int
	EmaPeriod2       int
	BbEnable         bool
	BbN              int
	BbK              float64
	IchimokuEnable   bool
	MacdEnable       bool
	MacdFastPeriod   int
	MacdSlowPeriod   int
	MacdSignalPeriod int
	RsiEnable        bool
	RsiPeriod        int
	RsiBuyThread     float64
	RsiSellThread    float64
}

type Ranking struct {
	Enable      bool
	Performance float64
}

/**
どのインディケータでトレードを行うかを返す
return TradeParams */
func (df *DataFrameCandle) OptimizeParams() *TradeParams {
	emaPerformance, emaPeriod1, emaPeriod2 := df.OptimizeEma()
	bbPerformance, bbN, bbK := df.OptimizeBb()
	macdPerformance, macdFastPeriod, macdSlowPeriod, macdSignalPeriod := df.OptimizeMacd()
	ichimokuPerforamcne := df.OptimizeIchimoku()
	rsiPerformance, rsiPeriod, rsiBuyThread, rsiSellThread := df.OptimizeRsi()

	emaRanking := &Ranking{false, emaPerformance}
	bbRanking := &Ranking{false, bbPerformance}
	macdRanking := &Ranking{false, macdPerformance}
	ichimokuRanking := &Ranking{false, ichimokuPerforamcne}
	rsiRanking := &Ranking{false, rsiPerformance}

	// Rankingに格納してソートする
	rankings := []*Ranking{emaRanking, bbRanking, macdRanking, ichimokuRanking, rsiRanking}
	sort.Slice(rankings, func(i, j int) bool { return rankings[i].Performance > rankings[j].Performance })

	// いずれのインディケータもfalseの場合再度キャンドルの結果を得てからオプティマイズし直す
	isEnable := false
	for i, ranking := range rankings {
		if i >= config.Config.NumRanking {
			break
		}
		if ranking.Performance > 0 {
			ranking.Enable = true
			// 1つでもインディケータが使えるならtrueにする
			isEnable = true
		}
	}
	// インディケータが1つも使えない場合はnilを返し再度オプティマイズさせる
	if !isEnable {
		return nil
	}

	// 環境変数から使用するインディケータを選出する
	envNumRanking := os.Getenv("NUM_RANKING")
	numRanking, _ := strconv.Atoi(envNumRanking)

	for i, ranking := range rankings {
		if i >= numRanking {
			break
		}
		if ranking.Performance > 0 {
			ranking.Enable = true
		}
	}

	tradeParams := &TradeParams{
		EmaEnable:        emaRanking.Enable,
		EmaPeriod1:       emaPeriod1,
		EmaPeriod2:       emaPeriod2,
		BbEnable:         bbRanking.Enable,
		BbN:              bbN,
		BbK:              bbK,
		IchimokuEnable:   ichimokuRanking.Enable,
		MacdEnable:       macdRanking.Enable,
		MacdFastPeriod:   macdFastPeriod,
		MacdSlowPeriod:   macdSlowPeriod,
		MacdSignalPeriod: macdSignalPeriod,
		RsiEnable:        rsiRanking.Enable,
		RsiPeriod:        rsiPeriod,
		RsiBuyThread:     rsiBuyThread,
		RsiSellThread:    rsiSellThread,
	}
	return tradeParams
}
