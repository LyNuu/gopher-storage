package pydio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type PydioStorage struct {
	client   *http.Client
	baseURL  string
	apiToken string
}

func NewPydioStorage(baseURL, apiToken string) *PydioStorage {
	return &PydioStorage{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		baseURL:  baseURL,
		apiToken: apiToken,
	}
}

func (p *PydioStorage) CreateUserFolder(ctx context.Context, userID uuid.UUID) error {
	targetPath := fmt.Sprintf("personal-files/%s", userID.String())

	requestBody := map[string]any{
		"Nodes": []map[string]any{
			{
				"Path":  targetPath,
				"Type":  "COLLECTION",
				"IsDir": true,
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal pydio request: %w", err)
	}
	url := fmt.Sprintf("%s/a/tree/nodes", p.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create http request to pydio: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiToken))

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("pydio api request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("pydio returned unexpected status: %d", resp.StatusCode)
	}
	return nil
}
