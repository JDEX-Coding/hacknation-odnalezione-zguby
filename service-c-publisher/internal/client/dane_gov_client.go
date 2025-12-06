package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hacknation/odnalezione-zguby/service-c-publisher/internal/models"
	"github.com/rs/zerolog/log"
)

// DaneGovClient handles communication with dane.gov.pl API
type DaneGovClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewDaneGovClient creates a new dane.gov.pl API client
func NewDaneGovClient(baseURL, apiKey string) *DaneGovClient {
	return &DaneGovClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// PublishDataset publishes a dataset to dane.gov.pl
func (c *DaneGovClient) PublishDataset(ctx context.Context, dataset *models.DatasetRequest) (*models.DatasetResponse, error) {
	url := fmt.Sprintf("%s/api/v1/datasets", c.baseURL)

	jsonData, err := json.Marshal(dataset)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal dataset: %w", err)
	}

	log.Debug().
		Str("url", url).
		Str("payload", string(jsonData)).
		Msg("Publishing dataset to dane.gov.pl")

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Error().
			Int("status", resp.StatusCode).
			Str("body", string(body)).
			Msg("dane.gov.pl API returned error")
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response models.DatasetResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	log.Info().
		Str("dataset_id", response.Data.ID).
		Str("url", response.Data.Attributes.URL).
		Msg("Successfully published dataset to dane.gov.pl")

	return &response, nil
}

// GetDataset retrieves a dataset by ID
func (c *DaneGovClient) GetDataset(ctx context.Context, datasetID string) (*models.DatasetResponse, error) {
	url := fmt.Sprintf("%s/api/v1/datasets/%s", c.baseURL, datasetID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response models.DatasetResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// HealthCheck checks if the API is available
func (c *DaneGovClient) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	return nil
}
