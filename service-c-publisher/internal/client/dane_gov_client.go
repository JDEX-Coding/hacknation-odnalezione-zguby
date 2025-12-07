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
	email      string
	password   string
	token      string
	httpClient *http.Client
}

// NewDaneGovClient creates a new dane.gov.pl API client
func NewDaneGovClient(baseURL, email, password string) *DaneGovClient {
	return &DaneGovClient{
		baseURL:  baseURL,
		email:    email,
		password: password,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Login authenticates with dane.gov.pl API and obtains JWT token
func (c *DaneGovClient) Login(ctx context.Context) error {
	url := fmt.Sprintf("%s/auth/login", c.baseURL)

	loginReq := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "login",
			"attributes": map[string]string{
				"email":    c.email,
				"password": c.password,
			},
		},
	}

	jsonData, err := json.Marshal(loginReq)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %w", err)
	}

	log.Debug().
		Str("url", url).
		Str("email", c.email).
		Msg("Logging in to dane.gov.pl")

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create login request for %s: %w", url, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send login request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read login response: %w", err)
	}

	// API returns 201 Created on successful login
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Error().
			Int("status", resp.StatusCode).
			Str("body", string(body)).
			Msg("Login failed")
		return fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON:API response format
	var loginResp struct {
		Data struct {
			Attributes struct {
				Token string `json:"token"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &loginResp); err != nil {
		return fmt.Errorf("failed to unmarshal login response: %w", err)
	}

	c.token = loginResp.Data.Attributes.Token

	log.Info().
		Str("email", c.email).
		Msg("✅ Successfully logged in to dane.gov.pl")

	return nil
}

// AddResource adds a resource to an existing dataset
func (c *DaneGovClient) AddResource(ctx context.Context, datasetID string, resource *models.ResourceRequest) (*models.ResourceResponse, error) {
	url := fmt.Sprintf("%s/api/1.4/datasets/%s/resources", c.baseURL, datasetID)

	jsonData, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resource: %w", err)
	}

	log.Debug().
		Str("url", url).
		Str("dataset_id", datasetID).
		Msg("Adding resource to dataset")

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))

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
			Msg("Failed to add resource")
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response models.ResourceResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	log.Info().
		Str("resource_id", response.Data.ID).
		Str("dataset_id", datasetID).
		Msg("Successfully added resource to dataset")

	return &response, nil
}
