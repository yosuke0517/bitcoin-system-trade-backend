package model

import (
	"app/domain"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

const (
	tableNameSignalEvents = "SIGNAL_EVENTS"
)

type SignalEvent struct {
	Time        time.Time `json:"time"`
	ProductCode string    `json:"product_code"`
	Side        string    `json:"side"`
	Price       float64   `json:"price"`
	Size        float64   `json:"size"`
	Atr         int       `json:"atr"`
	AtrRate     float64   `json:"atr_rate"`
	Pnl         float64   `json:"pnl"`
	ReOpen      bool      `json:"re_open"`
}

/** 売買のイベントを書き込む */
func (s *SignalEvent) Save() bool {
	tableName := tableNameSignalEvents
	cmd := fmt.Sprintf("INSERT INTO %s (time, product_code, side, price, size, atr, atr_rate, pnl, re_open) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", tableName)
	ins, err := domain.DB.Prepare(cmd)
	if err != nil {
		log.Println(err)
	}
	_, err = ins.Exec(s.Time, s.ProductCode, s.Side, s.Price, s.Size, s.Atr, s.AtrRate, s.Pnl, s.ReOpen)
	if err != nil {
		// 今回は同じ時間で複数売買させない
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			log.Println(err)
			return true
		}
		return false
	}
	return true
}

type SignalEvents struct {
	Signals []SignalEvent `json:"signals,omitempty"`
}

func NewSignalEvents() *SignalEvents {
	return &SignalEvents{}
}

// BUY SELL BUY SELL等の情報をlimitを指定して返却する
func GetSignalEventsByCount(loadEvents int) *SignalEvents {
	tableName := tableNameSignalEvents
	cmd := fmt.Sprintf(`SELECT * FROM (SELECT time, product_code, side, price, size, atr, atr_rate, pnl, re_open FROM %s WHERE product_code = ? ORDER BY time DESC LIMIT ? ) as events ORDER BY time ASC;`, tableName)
	rows, err := domain.DB.Query(cmd, os.Getenv("PRODUCT_CODE"), loadEvents)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer rows.Close()

	var signalEvents SignalEvents
	for rows.Next() {
		var signalEvent SignalEvent
		rows.Scan(&signalEvent.Time, &signalEvent.ProductCode, &signalEvent.Side, &signalEvent.Price, &signalEvent.Size, &signalEvent.Atr, &signalEvent.AtrRate, &signalEvent.Pnl, &signalEvent.ReOpen)
		signalEvents.Signals = append(signalEvents.Signals, signalEvent)
	}
	err = rows.Err()
	if err != nil {
		log.Println(err)
		return nil
	}
	return &signalEvents
}

// BUY SELL BUY SELL等の情報を全て取得する
func GetAllSignalEvents(backTest bool) *SignalEvents {
	tableName := tableNameSignalEvents
	cmd := fmt.Sprintf(`SELECT * FROM (SELECT time, product_code, side, price, size, atr, atr_rate, pnl, re_open FROM %s WHERE product_code = ? ORDER BY time DESC) as events ORDER BY time ASC;`, tableName)
	rows, err := domain.DB.Query(cmd, os.Getenv("PRODUCT_CODE"))
	if err != nil {
		log.Println(err)
		return nil
	}
	defer rows.Close()

	var signalEvents SignalEvents
	for rows.Next() {
		var signalEvent SignalEvent
		rows.Scan(&signalEvent.Time, &signalEvent.ProductCode, &signalEvent.Side, &signalEvent.Price, &signalEvent.Size, &signalEvent.Atr, &signalEvent.AtrRate, &signalEvent.Pnl, &signalEvent.ReOpen)
		signalEvents.Signals = append(signalEvents.Signals, signalEvent)
	}
	err = rows.Err()
	if err != nil {
		log.Println(err)
		return nil
	}
	return &signalEvents
}

/** 時間を指定して売買イベントの結果を取得する */
func GetSignalEventsAfterTime(timeTime time.Time) *SignalEvents {
	tableName := tableNameSignalEvents
	// MySqlの場合はサブクエリにasが必要
	cmd := fmt.Sprintf(`SELECT * FROM (SELECT time, product_code, side, price, size, atr, atr_rate, pnl, re_open FROM %s WHERE time >= ? ORDER BY time DESC) as events ORDER BY time ASC;`, tableName)
	rows, err := domain.DB.Query(cmd, timeTime)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer rows.Close()

	var signalEvents SignalEvents
	for rows.Next() {
		var signalEvent SignalEvent
		rows.Scan(&signalEvent.Time, &signalEvent.ProductCode, &signalEvent.Side, &signalEvent.Price, &signalEvent.Size, &signalEvent.Atr, &signalEvent.AtrRate, &signalEvent.Pnl, &signalEvent.ReOpen)
		signalEvents.Signals = append(signalEvents.Signals, signalEvent)
	}
	return &signalEvents
}

type Events struct {
	Length int `json:"length"`
}

/** 全ての売買イベント数を返す */
func GetAllSignalEventsCount() int {
	tableName := tableNameSignalEvents
	cmd := fmt.Sprintf(`SELECT count(*) FROM %s ;`, tableName)
	var events Events
	err := domain.DB.QueryRow(cmd).Scan(&events.Length)
	if err != nil {
		log.Println(err)
		return 0
	}
	eventsLength := events.Length
	return eventsLength
}

/** 買えるかどうかの判定 */
func (s *SignalEvents) CanBuy(time time.Time, reOpen bool) bool {
	lenSignals := len(s.Signals)
	if lenSignals == 0 {
		return true
	}
	// TODO ショート対応
	lastSignal := s.Signals[lenSignals-1]
	if reOpen {
		return true
	}
	// 同じ時間の場合はreOpenがtrueの時のみ許可する
	if (lastSignal.Side == "SELL" || lenSignals%2 == 0) && (lastSignal.Time.Before(time) || (lastSignal.Time.Equal(time))) {
		return true
	}
	return false
}

/** 売れるかどうかの判定 */
func (s *SignalEvents) CanSell(time time.Time, reOpen bool) bool {
	lenSignals := len(s.Signals)
	if lenSignals == 0 {
		return true
	}

	lastSignal := s.Signals[lenSignals-1]
	// TODO ショート対応
	if reOpen {
		return true
	}
	// 同じ時間の場合はreOpenがtrueの時のみ許可する
	if (lastSignal.Side == "BUY" || lenSignals%2 == 0) && (lastSignal.Time.Before(time) || (lastSignal.Time.Equal(time))) {
		return true
	}
	return false
}

/** 購入 */
func (s *SignalEvents) Buy(ProductCode string, time time.Time, price, size float64, save bool, reOpen bool, orderPrice float64, atr int, pnl float64, bbRate float64) bool {
	atrRate := 0.0
	canBuy := s.CanBuy(time, reOpen)
	if !canBuy {
		return false
	}
	if orderPrice > 0 {
		atrRate = (float64(atr) / orderPrice) * 100
	}
	signalEvent := SignalEvent{
		ProductCode: ProductCode,
		Time:        time,
		Side:        "BUY",
		Price:       price,
		Size:        size,
		Atr:         atr,
		AtrRate:     atrRate,
		Pnl:         pnl,
		ReOpen:      reOpen,
	}
	// バックテスト等でセーブしたくない場合があるためBackTestフラグが必要
	if save {
		log.Printf("イベントを保存します：%s", signalEvent.Side)
		signalEvent.Save()
	}
	s.Signals = append(s.Signals, signalEvent)
	return true
}

/** 売却 */
func (s *SignalEvents) Sell(productCode string, time time.Time, price, size float64, save bool, reOpen bool, orderPrice float64, atr int, pnl float64, bbRate float64) bool {
	atrRate := 0.0
	canSell := s.CanSell(time, reOpen)
	if !canSell {
		return false
	}
	if orderPrice > 0 {
		atrRate = (float64(atr) / orderPrice) * 100
	}

	signalEvent := SignalEvent{
		ProductCode: productCode,
		Time:        time,
		Side:        "SELL",
		Price:       price,
		Size:        size,
		Atr:         atr,
		AtrRate:     atrRate,
		Pnl:         pnl,
		ReOpen:      reOpen,
	}
	// バックテスト等でセーブしたくない場合があるためBackTestフラグが必要
	if save {
		log.Printf("イベントを保存します：%s", signalEvent.Side)
		signalEvent.Save()
	}
	s.Signals = append(s.Signals, signalEvent)
	return true
}

func (s *SignalEvents) Profit() float64 {
	total := 0.0
	beforeSell := 0.0
	isHolding := false
	// TODO ショート対応
	for i, signalEvent := range s.Signals {
		// ロングオープンはマイナス
		if i%2 == 0 && signalEvent.Side == "BUY" {
			total -= signalEvent.Price * signalEvent.Size
			isHolding = true
		}
		// ロングクローズでプラス
		if i%2 == 1 && signalEvent.Side == "SELL" {
			total += signalEvent.Price * signalEvent.Size
			isHolding = false
			beforeSell = total
		}
		// ショートオープンはプラス
		if i%2 == 0 && signalEvent.Side == "SELL" {
			total += signalEvent.Price * signalEvent.Size
			isHolding = true
		}
		// ショートクローズでマイナス
		if i%2 == 1 && signalEvent.Side == "BUY" {
			total -= signalEvent.Price * signalEvent.Size
			isHolding = false
			beforeSell = total
		}
	}
	if isHolding == true {
		return beforeSell
	}
	return total
}

/** jsonへマーシャル */
func (s SignalEvents) MarshalJSON() ([]byte, error) {
	value, err := json.Marshal(&struct {
		Signals []SignalEvent `json:"signals,omitempty"`
		Profit  float64       `json:"profit,omitempty"`
	}{
		Signals: s.Signals,
		Profit:  s.Profit(),
	})
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return value, err
}

/** バックテスト用（データベースじゃない場所から取得する）*/
func (s *SignalEvents) CollectAfter(time time.Time) *SignalEvents {
	for i, signal := range s.Signals {
		// timeがsignal.Timeより後の場合はスキップする
		if time.After(signal.Time) {
			continue
		}
		return &SignalEvents{Signals: s.Signals[i:]}
	}
	return nil
}
