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
		//"30s": 30 * time.Second,
		"1m": time.Minute,
		"5m": time.Minute * 5,
		"1h": time.Hour,
	}

	Config = ConfigList{
		Durations:        durations,
		TradeDuration:    durations[cfg.Section("gotrading").Key("trade_duration").String()],
		UsePercent:       cfg.Section("gotrading").Key("use_percent").MustFloat64(),
		BackTest:         cfg.Section("gotrading").Key("back_test").MustBool(),
		DataLimit:        cfg.Section("gotrading").Key("data_limit").MustInt(),
		StopLimitPercent: cfg.Section("gotrading").Key("stop_limit_percent").MustFloat64(),
		NumRanking:       cfg.Section("gotrading").Key("num_ranking").MustInt(),
	}
}
