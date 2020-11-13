package model

import (
	"app/domain/tradingalgo"
	"github.com/markcheno/go-talib"
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

type Macd struct {
	FastPeriod   int       `json:"fast_period,omitempty"`
	SlowPeriod   int       `json:"slow_period,omitempty"`
	SignalPeriod int       `json:"signal_period,omitempty"`
	Macd         []float64 `json:"macd,omitempty"`
	MacdSignal   []float64 `json:"macd_signal,omitempty"`
	MacdHist     []float64 `json:"macd_hist,omitempty"`
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
