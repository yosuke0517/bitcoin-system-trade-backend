package main

import (
	"app/application/controllers"
	"app/application/server"
	"app/utils"
	"github.com/joho/godotenv"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/sirupsen/logrus"
	"os"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		logrus.Fatal("Error loading .env")
	}
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.JSONFormatter{})
}

func main() {
	e := echo.New()

	//Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.CORS())

	utils.LoggingSettings(os.Getenv("LOG_FILE"))

	/**
	リアルタイム controllerから
	*/
	go controllers.StreamIngestionData()

	/**
	オーダー一覧 TODO 固定じゃなくて動的にする
	*/
	//i := "JRF20200710-160500-132315"
	//params := map[string]string{
	//	"product_code":              "FX_BTC_JPY",
	//	"child_order_acceptance_id": i,
	//}
	//r, _ := bitflyerClient.ListOrder(params) // TODO: 注文ないときは「""」で返ってくる

	//s := model.NewSignalEvents()
	//df, _ := service.GetAllCandle(os.Getenv("PRODUCT_CODE"), time.Minute, 10)
	//c1 := df.Candles[0]
	//c2 := df.Candles[5]
	//s.Buy(os.Getenv("PRODUCT_CODE"), c1.Time, c1.Close, 1.0, true)
	//s.Sell(os.Getenv("PRODUCT_CODE"), c2.Time, c2.Close, 1.0, true)
	//fmt.Println(model.GetSignalEventsByCount(1))
	//fmt.Println(model.GetSignalEventsAfterTime(c1.Time))
	//fmt.Println(s.CollectAfter(time.Now()))
	//fmt.Println(s.CollectAfter(c1.Time))
	server.Serve()
}
