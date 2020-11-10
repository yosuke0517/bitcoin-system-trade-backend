package service

import (
	"app/bitflyer"
	"fmt"
	"os"
)

// クローズが約定しているかのチェック（建玉を保有していれば false, 保有していなければ true）
func CloseOrderExecutionCheck() bool {

	bitflyerClient := bitflyer.New(os.Getenv("API_KEY"), os.Getenv("API_SECRET"))
	params := map[string]string{
		"product_code": "FX_BTC_JPY",
	}
	positionRes, _ := bitflyerClient.GetPositions(params)
	if len(positionRes) == 0 {
		fmt.Println("クローズオーダーなしのため取引可能")
		return true
	} else {
		fmt.Println("クローズオーダーありのため取引不可")
		return false
	}
}
