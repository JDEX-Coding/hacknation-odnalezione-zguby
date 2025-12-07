package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func main() {
	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
	})

	// Mock login endpoint
	http.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse login request
		var loginReq struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Simple validation (accept any non-empty credentials for testing)
		if loginReq.Email == "" || loginReq.Password == "" {
			http.Error(w, "Email and password required", http.StatusBadRequest)
			return
		}

		// Generate mock JWT token
		mockToken := "mock-jwt-token-" + uuid.New().String()[:8]

		response := map[string]interface{}{
			"token":         mockToken,
			"refresh_token": "mock-refresh-token-" + uuid.New().String()[:8],
			"user": map[string]interface{}{
				"id":    uuid.New().String(),
				"email": loginReq.Email,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)

		log.Printf("✅ Login successful: %s (token: %s)", loginReq.Email, mockToken)
	})

	// Mock dataset creation endpoint
	http.HandleFunc("/api/v1/datasets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse request body
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Extract title from request
		title := "Unknown Dataset"
		if data, ok := reqBody["data"].(map[string]interface{}); ok {
			if attrs, ok := data["attributes"].(map[string]interface{}); ok {
				if t, ok := attrs["title"].(string); ok {
					title = t
				}
			}
		}

		// Generate response
		datasetID := uuid.New().String()
		slug := "dataset-" + datasetID[:8]

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"id":   datasetID,
				"type": "dataset",
				"attributes": map[string]interface{}{
					"title":   title,
					"slug":    slug,
					"status":  "published",
					"url":     "http://localhost:8000/datasets/" + slug,
					"created": time.Now().Format(time.RFC3339),
				},
			},
			"links": map[string]string{
				"self": "http://localhost:8000/api/v1/datasets/" + datasetID,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)

		log.Printf("✅ Dataset created: %s - %s", datasetID, title)
	})

	// Mock dataset retrieval endpoint
	http.HandleFunc("/api/v1/datasets/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		datasetID := r.URL.Path[len("/api/v1/datasets/"):]

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"id":   datasetID,
				"type": "dataset",
				"attributes": map[string]interface{}{
					"title":  "Mock Dataset",
					"status": "published",
					"url":    "http://localhost:8000/datasets/" + datasetID,
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	log.Println("🚀 Mock dane.gov.pl API server starting on :8000")
	log.Println("📍 Health check: http://localhost:8000/health")
	log.Println("📍 API endpoint: http://localhost:8000/api/v1/datasets")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
