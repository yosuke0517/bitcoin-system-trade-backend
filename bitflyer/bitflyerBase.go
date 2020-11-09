/*
bitflyer is access to bitflyterAPI
*/
package bitflyer

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const baseURL = "https://api.bitflyer.com/v1/"

// TODO usecaces/dto/配下へファイルとして格納
type APIClient struct {
	key        string
	secret     string
	httpClient *http.Client
}

func New(key, secret string) *APIClient {
	bitflyerClient := &APIClient{key, secret, &http.Client{}}
	return bitflyerClient
}

// header returns the map[string]string
func (api APIClient) header(method, endpoint string, body []byte) map[string]string {
	timeStamp := strconv.FormatInt(time.Now().Unix(), 10)
	message := timeStamp + method + endpoint + string(body)

	mac := hmac.New(sha256.New, []byte(api.secret))
	mac.Write([]byte(message))
	sign := hex.EncodeToString(mac.Sum(nil))
	return map[string]string{
		"ACCESS-KEY":       api.key,
		"ACCESS-TIMESTAMP": timeStamp,
		"ACCESS-SIGN":      sign,
		"Content-Type":     "application/json",
	}
}

func (api *APIClient) doRequest(method, urlPath string, query map[string]string, data []byte) (body []byte, statusCode int, err error) {
	baseURL, err := url.Parse(baseURL)
	if err != nil {
		log.Fatal(err)
	}
	apiURL, err := url.Parse(urlPath)
	if err != nil {
		log.Fatal(err)
	}
	endPoint := baseURL.ResolveReference(apiURL).String()

	// リクエストを作る
	req, err := http.NewRequest(method, endPoint, bytes.NewBuffer(data))
	if err != nil {
		log.Fatal(err)
	}
	// クエリー
	q := req.URL.Query()
	for key, value := range query {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()

	for key, value := range api.header(method, req.URL.RequestURI(), data) {
		req.Header.Add(key, value)
	}
	// APIアクセス
	resp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, 500, err
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

/*
/v1/tickerのレスポンス
*/
type Ticker struct {
	ProductCode     string  `json:"product_code"`
	Timestamp       string  `json:"timestamp"`
	TickID          int     `json:"tick_id"`
	BestBid         float64 `json:"best_bid"`
	BestAsk         float64 `json:"best_ask"`
	BestBidSize     float64 `json:"best_bid_size"`
	BestAskSize     float64 `json:"best_ask_size"`
	TotalBidDepth   float64 `json:"total_bid_depth"`
	TotalAskDepth   float64 `json:"total_ask_depth"`
	Ltp             float64 `json:"ltp"`
	Volume          float64 `json:"volume"`
	VolumeByProduct float64 `json:"volume_by_product"`
}

/*
中間値を求める
*/
func (t *Ticker) GetMidPrice() float64 {
	return (t.BestBid + t.BestAsk) / 2
}

/*
データベースが対応している日付型になおすメソッド
*/
func (t *Ticker) DateTime() time.Time {
	dateTime, err := time.Parse(time.RFC3339, t.Timestamp)
	//jst, _ := time.LoadLocation("Asia/Tokyo")
	//now := dateTime.In(jst)
	if err != nil {
		log.Printf("action=DateTime, err=%s", err.Error())
	}
	return dateTime
}

/*
時間変換用メソッド
@param 時間単位：h,m,s
@return time.Time（12:12:00 → duration hを与えると12:00:00に変換される
*/
func (t *Ticker) TruncateDateTime(duration time.Duration) time.Time {
	return t.DateTime().Truncate(duration)
}
