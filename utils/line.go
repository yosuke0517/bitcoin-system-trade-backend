package utils

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// LINE APIのレスポンス
type Response struct {
	Message string `json:"message"`
}

// 引数で受け取った文字列をLINE通知する
func SendLine(result string) (*http.Response, error) {
	accessToken := os.Getenv("LINEnotyfyToken")
	msg := result

	URL := os.Getenv("LINEpostURL")
	u, err := url.ParseRequestURI(URL)
	if err != nil {
		log.Fatal(err)
	}

	c := &http.Client{}

	form := url.Values{}
	form.Add("message", msg)

	body := strings.NewReader(form.Encode())

	req, err := http.NewRequest("POST", u.String(), body)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	res, err := c.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	return res, nil
}
