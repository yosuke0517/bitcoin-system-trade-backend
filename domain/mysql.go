package domain

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"log"
	"os"
)

var DB *sql.DB

func init() {
	envErr := godotenv.Load()
	if envErr != nil {
		logrus.Fatal("Error loading .env")
	}
	var err error
	/**
	  Memo:"DB_HOST"はdockerの場合データベースコンテナ名
	*/
	DB, err = sql.Open("mysql", os.Getenv("DB_USERNAME")+":"+os.Getenv("DB_PASSWORD")+
		"@tcp("+os.Getenv("DB_HOST")+":"+os.Getenv("DB_PORT")+")/"+
		os.Getenv("DB_DATABASE")+
		"?charset=utf8mb4&parseTime=true")

	if err != nil {
		log.Fatal(err)
	}
}
