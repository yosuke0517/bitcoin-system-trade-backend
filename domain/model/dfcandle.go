package model

import (
	"github.com/markcheno/go-talib"
	"time"
)

type DataFrameCandle struct {
	ProductCode string        `json:"product_code"`
	Duration    time.Duration `json:"duration"`
	Candles     []Candle      `json:"candles"`
	Smas        []Sma         `json:"smas,omitempty"`
	Emas        []Ema         `json:"emas,omitempty"`
	BBands      *BBands       `json:"bbands,omitempty"` // スライスじゃない場合はポインタ（ポインタがないとStructの空のjsonを返してしまいomitemptyが効かない）
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
