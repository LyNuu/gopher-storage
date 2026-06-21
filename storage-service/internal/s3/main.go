package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	// Адрес вашего Pydio Cells сервера
	cellsURL = "https://localhost:8070"

	// Personal Access Token (используется как S3 Access Key)
	accessToken = "xQsDXYOrP7dYLOn_tiAODmPAWxqSw6UtuK4KFUyij6A.Y4wRXNDFOw9keU8GzBiM4Qwvnf3hMQwYSy62kH51N7M"

	// Фиксированный S3 Secret Key для gateway Pydio Cells
	gatewaySecret = "gatewaysecret"

	// Bucket в Cells всегда называется "io"
	bucket = "io"

	// Слаг workspace + имя папки (буквально путь, как в breadcrumb веб-интерфейса)
	folderPath = "personal-files/57b88e12-4701-4579-b347-9ed9e3b96366"

	// Локальный файл для загрузки
	localFile = "./go-way.jpg"

	// Имя файла после загрузки на сервер
	remoteFileName = "go-way.jpg"
)

func main() {
	// localhost / самоподписанный сертификат -> отключаем проверку TLS
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	s3Client := s3.New(s3.Options{
		BaseEndpoint: aws.String(cellsURL),
		Region:       "us-east-1", // Cells регион не использует, но AWS SDK требует значение
		UsePathStyle: true,
		HTTPClient:   httpClient,
		Credentials:  credentials.NewStaticCredentialsProvider(accessToken, gatewaySecret, ""),
	})

	file, err := os.Open(localFile)
	if err != nil {
		log.Fatalf("Не удалось открыть файл: %v", err)
	}
	defer file.Close()

	objectKey := fmt.Sprintf("%s/%s", folderPath, remoteFileName)

	ctx := context.Background()
	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
		Body:   file,
	})
	if err != nil {
		log.Fatalf("Ошибка при загрузке: %v", err)
	}

	fmt.Printf("🎉 Успех! Файл загружен в %s\n", objectKey)
}
