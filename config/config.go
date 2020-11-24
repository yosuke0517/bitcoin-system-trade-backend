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
		//"1s":  time.Second,
		//"15s": 15 * time.Second,
		"30s": time.Second * 30,
		"1m":  time.Minute,
		"5m":  time.Minute * 5,
		"1h":  time.Hour,
	}

	Config = ConfigList{
		Durations:        durations,
		TradeDuration:    durations[cfg.Section("gotrade").Key("trade_duration").String()],
		UsePercent:       cfg.Section("gotrade").Key("use_percent").MustFloat64(),
		BackTest:         cfg.Section("gotrade").Key("back_test").MustBool(),
		DataLimit:        cfg.Section("gotrade").Key("data_limit").MustInt(),
		StopLimitPercent: cfg.Section("gotrade").Key("stop_limit_percent").MustFloat64(),
		NumRanking:       cfg.Section("gotrade").Key("num_ranking").MustInt(),
	}
}
