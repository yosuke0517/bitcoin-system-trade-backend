package config

import (
	"gopkg.in/ini.v1"
	"log"
	"os"
	"time"
)

type ConfigList struct {
	ApiKey           string
	ApiSecret        string
	LogFile          string
	ProductCode      string
	TradeDuration    string
	TradeSuffix      string
	Durations        map[string]time.Duration
	DbName           string
	DbHost           string
	DbPass           string
	DbUserName       string
	DbPort           string
	SQLDriver        string
	Port             int
	BackTest         bool
	UsePercent       float64
	DataLimit        int
	StopLimitPercent float64
	NumRanking       int
	Continue         bool
	OpenableBbRate   float64
	OpenableBbWith   float64
	AwsAccessKey     string
	AwsSecretKey     string
	IsProduction     bool
	CandleLengthMin  int
	LineNotifyToken  string
	LinePostUrl      string
	BacketName       string
}

var Config ConfigList

func init() {
	cfg, err := ini.Load("config.ini")
	if err != nil {
		log.Printf("Failed to read file: %v", err)
		// エラーコード１で終了
		os.Exit(1)
	}
	durations := map[string]time.Duration{
		"15m": time.Minute * 15,
		"30m": time.Minute * 30,
		"1h":  time.Hour,
	}

	Config = ConfigList{
		ApiKey:           cfg.Section("bitflyer").Key("api_key").String(),
		ApiSecret:        cfg.Section("bitflyer").Key("api_secret").String(),
		LogFile:          cfg.Section("gotrade").Key("log_file").String(),
		ProductCode:      cfg.Section("gotrade").Key("product_code").String(),
		TradeDuration:    cfg.Section("gotrade").Key("trade_duration").String(),
		TradeSuffix:      cfg.Section("gotrade").Key("trade_suffix").String(),
		Durations:        durations,
		DbName:           cfg.Section("db").Key("db_name").String(),
		DbHost:           cfg.Section("db").Key("host").String(),
		DbPass:           cfg.Section("db").Key("password").String(),
		DbUserName:       cfg.Section("db").Key("user_name").String(),
		DbPort:           cfg.Section("db").Key("port").String(),
		BackTest:         cfg.Section("gotrade").Key("back_test").MustBool(),
		UsePercent:       cfg.Section("gotrade").Key("use_percent").MustFloat64(),
		DataLimit:        cfg.Section("gotrade").Key("data_limit").MustInt(),
		StopLimitPercent: cfg.Section("gotrade").Key("stop_limit_percent").MustFloat64(),
		NumRanking:       cfg.Section("gotrade").Key("num_ranking").MustInt(),
		Continue:         cfg.Section("gotrade").Key("continue").MustBool(),
		OpenableBbRate:   cfg.Section("gotrade").Key("openable_bb_rate").MustFloat64(),
		OpenableBbWith:   cfg.Section("gotrade").Key("openable_bb_with"),
		AwsAccessKey:     cfg.Section("aws").Key("access_key").String(),
		AwsSecretKey:     cfg.Section("aws").Key("secret_key").String(),
		IsProduction:     cfg.Section("gotrade").Key("production").MustBool(),
		CandleLengthMin:  cfg.Section("gotrade").Key("candle_length_min").MustInt(),
		LineNotifyToken:  cfg.Section("line").Key("notify_token").String(),
		LinePostUrl:      cfg.Section("line").Key("post_url").String(),
		BacketName:       cfg.Section("aws").Key("backet_name").String(),
	}
}
