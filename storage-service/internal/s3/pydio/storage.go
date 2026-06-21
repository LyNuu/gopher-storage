package pydio

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

const rootFolder = "personal-files"

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
		BaseEndpoint:               aws.String(baseURL),
		Region:                     "us-east-1",
		UsePathStyle:               true,
		HTTPClient:                 httpClient,
		Credentials:                credentials.NewStaticCredentialsProvider(accessToken, gatewaySecret, ""),
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
	})

	return &PydioStorage{
		s3Client:    s3Client,
		httpClient:  httpClient,
		baseURL:     baseURL,
		accessToken: accessToken,
		bucket:      "io",
	}
}

func (p *PydioStorage) objectKey(storageID uuid.UUID, fileName string) string {
	return fmt.Sprintf("%s/%s/%s", rootFolder, storageID.String(), fileName)
}

func (p *PydioStorage) CreateStorageFolder(ctx context.Context, storageID uuid.UUID) error {
	folderPath := fmt.Sprintf("%s/%s", rootFolder, storageID.String())

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

func (p *PydioStorage) UploadFile(ctx context.Context, storageID uuid.UUID, fileName string, body io.Reader, size int64, contentType string) error {
	_, err := p.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(p.bucket),
		Key:           aws.String(p.objectKey(storageID, fileName)),
		Body:          body,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("s3 upload failed: %w", err)
	}
	return nil
}

func (p *PydioStorage) DownloadFile(ctx context.Context, storageID uuid.UUID, fileName string) (io.ReadCloser, int64, string, error) {
	output, err := p.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(p.objectKey(storageID, fileName)),
	})
	if err != nil {
		return nil, 0, "", err
	}
	return output.Body, aws.ToInt64(output.ContentLength), aws.ToString(output.ContentType), nil
}

func (p *PydioStorage) DeleteFile(ctx context.Context, storageID uuid.UUID, fileName string) error {
	_, err := p.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(p.objectKey(storageID, fileName)),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file from s3: %w", err)
	}
	return nil
}

func (p *PydioStorage) CreateUploadLink(ctx context.Context, storageID uuid.UUID, fileName string, expires time.Duration) (string, error) {
	presigner := s3.NewPresignClient(p.s3Client)

	req, err := presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(p.objectKey(storageID, fileName)),
	}, s3.WithPresignExpires(expires))
	if err != nil {
		return "", fmt.Errorf("failed to presign upload url: %w", err)
	}
	return req.URL, nil
}
