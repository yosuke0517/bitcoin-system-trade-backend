package domain

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/ini.v1"
	"log"
	"os"
)

var DB *sql.DB

func init() {
	cfg, err := ini.Load("config.ini")
	if err != nil {
		log.Printf("Failed to read file: %v", err)
		// エラーコード１で終了
		os.Exit(1)
	}
	/**
	  Memo:"DB_HOST"はdockerの場合データベースコンテナ名
	*/
	DB, err = sql.Open("mysql", cfg.Section("db").Key("user_name").String()+":"+cfg.Section("db").Key("password").String()+
		"@tcp("+cfg.Section("db").Key("host").String()+":"+cfg.Section("db").Key("port").String()+")/"+
		cfg.Section("db").Key("db_name").String()+
		"?charset=utf8mb4&parseTime=true")

	if err != nil {
		log.Fatal(err)
	}
}
