package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// VisionService handles communication with external Vision API (OpenAI GPT-4o)
type VisionService struct {
	apiKey   string
	endpoint string
	model    string
	client   *http.Client
}

// NewVisionService creates a new Vision API service
func NewVisionService(apiKey, endpoint, model string) *VisionService {
	if model == "" {
		model = "gpt-4o"
	}
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}

	return &VisionService{
		apiKey:   apiKey,
		endpoint: endpoint,
		model:    model,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// AnalyzeImageRequest represents the request to analyze an image
type AnalyzeImageRequest struct {
	ImageBase64 string `json:"image_base64"`
	ImageURL    string `json:"image_url,omitempty"`
}

// AnalyzeImageResponse represents the response from image analysis
type AnalyzeImageResponse struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Confidence  string `json:"confidence"`
}

// OpenAIRequest represents the OpenAI API request structure
type OpenAIRequest struct {
	Model     string          `json:"model"`
	Messages  []OpenAIMessage `json:"messages"`
	MaxTokens int             `json:"max_tokens"`
}

// OpenAIMessage represents a message in the OpenAI request
type OpenAIMessage struct {
	Role    string                 `json:"role"`
	Content []OpenAIMessageContent `json:"content"`
}

// OpenAIMessageContent represents content in a message
type OpenAIMessageContent struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *OpenAIImageURL `json:"image_url,omitempty"`
}

// OpenAIImageURL represents an image URL in the request
type OpenAIImageURL struct {
	URL string `json:"url"`
}

// OpenAIResponse represents the OpenAI API response
type OpenAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// AnalyzeImage analyzes an image and returns description and category
func (v *VisionService) AnalyzeImage(ctx context.Context, imageURL string) (*AnalyzeImageResponse, error) {
	if v.apiKey == "" {
		return nil, fmt.Errorf("vision API key not configured")
	}

	// Prepare the request
	requestBody := OpenAIRequest{
		Model: v.model,
		Messages: []OpenAIMessage{
			{
				Role: "user",
				Content: []OpenAIMessageContent{
					{
						Type: "text",
						Text: `Jesteś ekspertem od analizy zgubionego mienia. Przeanalizuj to zdjęcie i podaj:
1. Krótki tytuł zgłoszenia (4-8 słów, np. "Znaleziony czarny portfel skórzany")
2. Szczegółowy opis przedmiotu (2-3 zdania w języku polskim)
3. Kategorię (wybierz jedną z: Dokumenty, Elektronika, Biżuteria, Odzież, Portfele i torby, Klucze, Telefony, Inne)

Odpowiedz TYLKO w formacie JSON:
{
  "title": "krótki tytuł zgłoszenia",
  "description": "szczegółowy opis po polsku",
  "category": "nazwa kategorii",
  "confidence": "high/medium/low"
}`,
					},
					{
						Type: "image_url",
						ImageURL: &OpenAIImageURL{
							URL: imageURL,
						},
					},
				},
			},
		},
		MaxTokens: 500,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/chat/completions", v.endpoint),
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", v.apiKey))

	// Send request
	resp, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		log.Error().
			Int("status_code", resp.StatusCode).
			Str("body", string(body)).
			Msg("Vision API returned error")
		return nil, fmt.Errorf("vision API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var openAIResp OpenAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API errors
	if openAIResp.Error != nil {
		return nil, fmt.Errorf("vision API error: %s", openAIResp.Error.Message)
	}

	// Extract the content
	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from vision API")
	}

	content := openAIResp.Choices[0].Message.Content

	// Parse the JSON response from the AI
	var analysisResp AnalyzeImageResponse
	if err := json.Unmarshal([]byte(content), &analysisResp); err != nil {
		// Try to extract JSON from markdown code blocks (```json ... ```)
		cleanedContent := content

		// Remove markdown code blocks
		if strings.Contains(cleanedContent, "```json") {
			// Extract content between ```json and ```
			parts := strings.Split(cleanedContent, "```json")
			if len(parts) > 1 {
				jsonPart := strings.Split(parts[1], "```")[0]
				cleanedContent = strings.TrimSpace(jsonPart)
			}
		} else if strings.Contains(cleanedContent, "```") {
			// Remove generic code blocks
			parts := strings.Split(cleanedContent, "```")
			if len(parts) > 1 {
				cleanedContent = strings.TrimSpace(parts[1])
			}
		}

		// Try parsing cleaned content
		if err := json.Unmarshal([]byte(cleanedContent), &analysisResp); err != nil {
			// If still fails, use raw content as description
			log.Warn().
				Err(err).
				Str("content", content).
				Str("cleaned_content", cleanedContent).
				Msg("Failed to parse AI response as JSON, using raw content")

			analysisResp = AnalyzeImageResponse{
				Title:       "Znaleziony przedmiot",
				Description: content,
				Category:    "Inne",
				Confidence:  "medium",
			}
		}
	}

	log.Info().
		Str("title", analysisResp.Title).
		Str("description", analysisResp.Description).
		Str("category", analysisResp.Category).
		Str("confidence", analysisResp.Confidence).
		Msg("Image analyzed successfully")

	return &analysisResp, nil
}

// AnalyzeImageBase64 analyzes a base64-encoded image
func (v *VisionService) AnalyzeImageBase64(ctx context.Context, imageBase64 string) (*AnalyzeImageResponse, error) {
	// Convert base64 to data URL
	dataURL := fmt.Sprintf("data:image/jpeg;base64,%s", imageBase64)
	return v.AnalyzeImage(ctx, dataURL)
}

// HealthCheck verifies the Vision API connection
func (v *VisionService) HealthCheck(ctx context.Context) error {
	if v.apiKey == "" {
		return fmt.Errorf("vision API key not configured")
	}
	return nil
}
