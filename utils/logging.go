package utils

import (
	"app/config"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io"
	"log"
	"os"
	"time"
)

func LoggingSettings(logFile string) {
	logfile, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("file=logFile err=%s", err.Error())
	}
	multiLogFile := io.MultiWriter(os.Stdout, logfile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetOutput(multiLogFile)
}

func UploadLogFile() {
	creds := credentials.NewStaticCredentials(config.Config.AwsAccessKey, config.Config.AwsSecretKey, "")
	// sessionの作成
	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: creds,
		Region:      aws.String("ap-northeast-1"),
	}))

	// ファイルを開く
	targetFilePath := "./debug.log"
	file, err := os.Open(targetFilePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	day := time.Now()
	fileNameSuffix := day.Format("2006-01-02")

	objectKey := fileNameSuffix

	// Uploaderを作成し、ローカルファイルをアップロード
	uploader := s3manager.NewUploader(sess)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(config.Config.BacketName),
		Key:    aws.String(objectKey),
		Body:   file,
	})
	if err != nil {
		log.Fatal(err)
	}
	removeErr := os.Remove("./debug.log")
	if err != nil {
		log.Fatal(removeErr)
	}
	LoggingSettings(config.Config.LogFile)
	log.Println("done")
}
