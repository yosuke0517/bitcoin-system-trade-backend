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
	tableNameSignalEvents         = "SIGNAL_EVENTS"
	tableNameSignalEventsBackTest = "SIGNAL_EVENTS_BACK_TEST"
)

type SignalEvent struct {
	Time        time.Time `json:"time"`
	ProductCode string    `json:"product_code"`
	Side        string    `json:"side"`
	Price       float64   `json:"price"`
	Size        float64   `json:"size"`
}

/** 売買のイベントを書き込む */
func (s *SignalEvent) Save(BackTest bool) bool {
	tableName := tableNameSignalEvents
	if BackTest {
		tableName = tableNameSignalEventsBackTest
	}
	cmd := fmt.Sprintf("INSERT INTO %s (time, product_code, side, price, size) VALUES (?, ?, ?, ?, ?)", tableName)
	ins, err := domain.DB.Prepare(cmd)
	if err != nil {
		log.Println(err)
	}
	_, err = ins.Exec(s.Time, s.ProductCode, s.Side, s.Price, s.Size)
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
	cmd := fmt.Sprintf(`SELECT * FROM (SELECT time, product_code, side, price, size FROM %s WHERE product_code = ? ORDER BY time DESC LIMIT ? ) as events ORDER BY time ASC;`, tableNameSignalEvents)
	rows, err := domain.DB.Query(cmd, os.Getenv("PRODUCT_CODE"), loadEvents)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer rows.Close()

	var signalEvents SignalEvents
	for rows.Next() {
		var signalEvent SignalEvent
		rows.Scan(&signalEvent.Time, &signalEvent.ProductCode, &signalEvent.Side, &signalEvent.Price, &signalEvent.Size)
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
	// MySqlの場合はサブクエリにasが必要
	cmd := fmt.Sprintf(`SELECT * FROM (SELECT time, product_code, side, price, size FROM %s WHERE time >= ? ORDER BY time DESC) as events ORDER BY time ASC;`, tableNameSignalEvents)
	rows, err := domain.DB.Query(cmd, timeTime)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer rows.Close()

	var signalEvents SignalEvents
	for rows.Next() {
		var signalEvent SignalEvent
		rows.Scan(&signalEvent.Time, &signalEvent.ProductCode, &signalEvent.Side, &signalEvent.Price, &signalEvent.Size)
		signalEvents.Signals = append(signalEvents.Signals, signalEvent)
	}
	return &signalEvents
}

type Events struct {
	Length int `json:"length"`
}

/** 全ての売買イベント数を返す */
func GetAllSignalEvents() int {
	cmd := fmt.Sprintf(`SELECT count(*) FROM %s ;`, tableNameSignalEvents)
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
func (s *SignalEvents) CanBuy(time time.Time) bool {
	lenSignals := len(s.Signals)
	if lenSignals == 0 {
		return true
	}

	lastSignal := s.Signals[lenSignals-1]
	// SignalEventsの最後がSELL, 最後のシグナルより後の時間であるかのチェック
	if lastSignal.Side == "SELL" && lastSignal.Time.Before(time) {
		return true
	}
	return false
}

/** 売れるかどうかの判定 */
func (s *SignalEvents) CanSell(time time.Time) bool {
	lenSignals := len(s.Signals)
	if lenSignals == 0 {
		return false
	}

	lastSignal := s.Signals[lenSignals-1]
	// SignalEventsの最後がSELL, 最後のシグナルより後の時間であるかのチェック
	if lastSignal.Side == "BUY" && lastSignal.Time.Before(time) {
		return true
	}
	return false
}

/** 購入 */
func (s *SignalEvents) Buy(ProductCode string, time time.Time, price, size float64, BackTest bool) bool {
	if !s.CanBuy(time) {
		return false
	}
	signalEvent := SignalEvent{
		ProductCode: ProductCode,
		Time:        time,
		Side:        "BUY",
		Price:       price,
		Size:        size,
	}
	// バックテスト等でセーブしたくない場合があるためsaveフラグが必要
	signalEvent.Save(BackTest)
	s.Signals = append(s.Signals, signalEvent)
	return true
}

/** 売却 */
func (s *SignalEvents) Sell(productCode string, time time.Time, price, size float64, BackTest bool) bool {

	if !s.CanSell(time) {
		return false
	}

	signalEvent := SignalEvent{
		ProductCode: productCode,
		Time:        time,
		Side:        "SELL",
		Price:       price,
		Size:        size,
	}
	// バックテスト等でセーブしたくない場合があるためsaveフラグが必要
	signalEvent.Save(BackTest)
	s.Signals = append(s.Signals, signalEvent)
	return true
}

func (s *SignalEvents) Profit() float64 {
	total := 0.0
	beforeSell := 0.0
	isHolding := false

	// イベントの数が奇数のときは保有判定
	if len(s.Signals)%1 == 0 {
		isHolding = true
	}

	for i, signalEvent := range s.Signals {
		if i == 0 && signalEvent.Side == "SELL" {
			continue
		}
		if (signalEvent.Side == "BUY" || signalEvent.Side == "SELL") && isHolding == false {
			total += signalEvent.Price * signalEvent.Size
			beforeSell = total
		}
		if (signalEvent.Side == "BUY" || signalEvent.Side == "SELL") && isHolding == true {
			total -= signalEvent.Price * signalEvent.Size
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
