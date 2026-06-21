package pydio

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"io"
	"net/http"
	"time"
)

type PydioStorage struct {
	s3Client    *s3.Client
	httpClient  *http.Client
	baseURL     string
	accessToken string
	bucket      string
}

func NewPydioStorage(baseURL, accessToken, gatewaySecret string) *PydioStorage {
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	s3Client := s3.New(s3.Options{
		BaseEndpoint: aws.String(baseURL),
		Region:       "us-east-1", // Cells регион не использует, но AWS SDK требует значение
		UsePathStyle: true,
		HTTPClient:   httpClient,
		Credentials:  credentials.NewStaticCredentialsProvider(accessToken, gatewaySecret, ""),
	})

	return &PydioStorage{
		s3Client:    s3Client,
		httpClient:  httpClient,
		baseURL:     baseURL,
		accessToken: accessToken,
		bucket:      "io",
	}
}

// CreateUserFolder создаёт настоящую папку (узел типа COLLECTION) через Tree Service REST API.
// Через S3 PutObject с пустым телом и "/" в конце ключа Pydio Cells не создаёт реальную папку
// (в отличие от "чистого" S3/MinIO) - там нужен явный вызов Tree API.
func (p *PydioStorage) CreateUserFolder(ctx context.Context, userID uuid.UUID) error {
	folderPath := fmt.Sprintf("personal-files/%s", userID.String())

	body, err := json.Marshal(map[string]interface{}{
		"Nodes": []map[string]string{
			{
				"Path": folderPath,
				"Type": "COLLECTION",
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal create folder request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/a/tree/create", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build create folder request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create folder in pydio: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("failed to create folder in pydio: unexpected status %d", resp.StatusCode)
	}

	return nil
}

func (p *PydioStorage) UploadUserFile(ctx context.Context, userID uuid.UUID, fileName string, fileStream io.Reader, fileSize int64, contentType string) error {
	targetKey := fmt.Sprintf("personal-files/%s/%s", userID.String(), fileName)

	_, err := p.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(p.bucket),
		Key:           aws.String(targetKey),
		Body:          fileStream,
		ContentLength: aws.Int64(fileSize),
		ContentType:   aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("s3 upload failed: %w", err)
	}
	return nil
}

func (p *PydioStorage) DownloadUserFile(ctx context.Context, userID uuid.UUID, fileName string) (io.ReadCloser, int64, string, error) {
	targetKey := fmt.Sprintf("personal-files/%s/%s", userID.String(), fileName)

	output, err := p.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(targetKey),
	})
	if err != nil {
		return nil, 0, "", err
	}

	// Возвращаем поток, размер и тип контента
	return output.Body, *output.ContentLength, *output.ContentType, nil
}

func (p *PydioStorage) DeleteUserFile(ctx context.Context, userID uuid.UUID, fileName string) error {
	targetKey := fmt.Sprintf("personal-files/%s/%s", userID.String(), fileName)

	_, err := p.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(targetKey),
	})

	if err != nil {
		return fmt.Errorf("failed to delete file from S3: %w", err)
	}
	return nil
}

func (p *PydioStorage) GetShareableLink(ctx context.Context, userID uuid.UUID, fileName string, duration time.Duration) (string, error) {
	targetKey := fmt.Sprintf("personal-files/%s/%s", userID.String(), fileName)

	presignClient := s3.NewPresignClient(p.s3Client)

	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(targetKey),
	}, s3.WithPresignExpires(duration))

	if err != nil {
		return "", fmt.Errorf("failed to presign url: %w", err)
	}

	return request.URL, nil
}

func (p *PydioStorage) CreateUploadLink(
	ctx context.Context,
	userID uuid.UUID,
	fileName string,
	expires time.Duration,
) (string, error) {

	key := fmt.Sprintf(
		"personal-files/%s/%s",
		userID.String(),
		fileName,
	)

	presigner := s3.NewPresignClient(p.s3Client)

	req, err := presigner.PresignPutObject(
		ctx,
		&s3.PutObjectInput{
			Bucket: aws.String(p.bucket),
			Key:    aws.String(key),
		},
		s3.WithPresignExpires(expires),
	)
	if err != nil {
		return "", err
	}

	return req.URL, nil
}
