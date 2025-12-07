package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// DatasetPublishRequest represents a request to publish a dataset to dane.gov.pl
type DatasetPublishRequest struct {
	DatasetID string `json:"dataset_id"`
}

// PublishDatasetHandler handles requests to publish a dataset to dane.gov.pl
func (h *Handler) PublishDatasetHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get dataset ID from URL or request body
	vars := mux.Vars(r)
	datasetID := vars["id"]

	// If not in URL, try request body
	if datasetID == "" {
		var req DatasetPublishRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		datasetID = req.DatasetID
	}

	if datasetID == "" {
		http.Error(w, "Dataset ID is required", http.StatusBadRequest)
		return
	}

	log.Info().
		Str("dataset_id", datasetID).
		Msg("📤 Publishing dataset to dane.gov.pl")

	// Fetch dataset from database
	dataset, found := h.items.GetDataset(datasetID)
	if !found {
		log.Error().
			Str("dataset_id", datasetID).
			Msg("Dataset not found")
		http.Error(w, "Dataset not found", http.StatusNotFound)
		return
	}

	log.Info().
		Str("dataset_id", datasetID).
		Str("title", dataset.Title).
		Msg("✅ Dataset retrieved from database")

	// Create event payload matching DatasetPublishEvent struct
	event := map[string]interface{}{
		"dataset_id":       dataset.ID,
		"title":            dataset.Title,
		"notes":            dataset.Notes,
		"url":              dataset.URL,
		"institution_name": dataset.InstitutionName,
		"email":            dataset.Email,
		"categories":       dataset.Categories,
		"tags":             dataset.Tags,
		"timestamp":        time.Now(), // Send as time.Time, not string
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal dataset event")
		http.Error(w, "Failed to process dataset", http.StatusInternalServerError)
		return
	}

	// Publish to RabbitMQ with routing key for dataset publication
	if err := h.rabbitMQ.PublishWithKey(ctx, eventJSON, "dataset.publish"); err != nil {
		log.Error().Err(err).Msg("Failed to publish dataset to RabbitMQ")
		http.Error(w, "Failed to publish dataset", http.StatusInternalServerError)
		return
	}

	log.Info().
		Str("dataset_id", datasetID).
		Str("routing_key", "dataset.publish").
		Msg("✅ Dataset published to RabbitMQ for dane.gov.pl publication")

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"dataset_id": datasetID,
		"message":    "Dataset queued for publication to dane.gov.pl",
		"status":     "pending",
	})
}

// PublishDatasetFormHandler renders a form to publish a dataset (HTMX version)
func (h *Handler) PublishDatasetFormHandler(w http.ResponseWriter, r *http.Request) {
	// Get dataset ID from URL
	vars := mux.Vars(r)
	datasetID := vars["id"]

	if datasetID == "" {
		http.Error(w, "Dataset ID is required", http.StatusBadRequest)
		return
	}

	// Fetch dataset from database
	dataset, found := h.items.GetDataset(datasetID)
	if !found {
		http.Error(w, "Dataset not found", http.StatusNotFound)
		return
	}

	// Render template with dataset info
	tmpl, ok := h.templates["dataset-publish.html"]
	if !ok {
		log.Error().Msg("Template dataset-publish.html not found")
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":   "Publikuj Dataset",
		"Dataset": dataset,
	}

	tmpl.ExecuteTemplate(w, "base.html", data)
}

// ListDatasetsAPIHandler returns all datasets as JSON
func (h *Handler) ListDatasetsAPIHandler(w http.ResponseWriter, r *http.Request) {
	datasets, err := h.items.ListDatasets()
	if err != nil {
		log.Error().Err(err).Msg("Failed to list datasets")
		http.Error(w, "Failed to list datasets", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"datasets": datasets,
		"count":    len(datasets),
	})
}

// GetDatasetAPIHandler returns a single dataset as JSON
func (h *Handler) GetDatasetAPIHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	datasetID := vars["id"]

	if datasetID == "" {
		http.Error(w, "Dataset ID is required", http.StatusBadRequest)
		return
	}

	dataset, found := h.items.GetDataset(datasetID)
	if !found {
		http.Error(w, "Dataset not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"dataset": dataset,
	})
}

// GetDatasetWithItemsAPIHandler returns a dataset with all its items as JSON
func (h *Handler) GetDatasetWithItemsAPIHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	datasetID := vars["id"]

	if datasetID == "" {
		http.Error(w, "Dataset ID is required", http.StatusBadRequest)
		return
	}

	datasetWithItems, err := h.items.GetDatasetWithItems(datasetID)
	if err != nil {
		log.Error().Err(err).Str("dataset_id", datasetID).Msg("Failed to get dataset with items")
		http.Error(w, fmt.Sprintf("Failed to get dataset: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"dataset": datasetWithItems,
	})
}
