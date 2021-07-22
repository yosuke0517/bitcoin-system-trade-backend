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
	TradeDuration    time.Duration
	Durations        map[string]time.Duration
	DbName           string
	SQLDriver        string
	Port             int
	BackTest         bool
	UsePercent       float64
	DataLimit        int
	StopLimitPercent float64
	NumRanking       int
	Continue         bool
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
		"5m":  time.Minute * 5,
		"15m": time.Minute * 15,
		"30m": time.Minute * 30,
		"1h":  time.Hour,
	}

	Config = ConfigList{
		Durations:        durations,
		ProductCode:      cfg.Section("gotrade").Key("product_code").String(),
		TradeDuration:    durations[cfg.Section("gotrade").Key("trade_duration").String()],
		UsePercent:       cfg.Section("gotrade").Key("use_percent").MustFloat64(),
		BackTest:         cfg.Section("gotrade").Key("back_test").MustBool(),
		DataLimit:        cfg.Section("gotrade").Key("data_limit").MustInt(),
		StopLimitPercent: cfg.Section("gotrade").Key("stop_limit_percent").MustFloat64(),
		NumRanking:       cfg.Section("gotrade").Key("num_ranking").MustInt(),
		Continue:         cfg.Section("gotrade").Key("continue").MustBool(),
	}
}
