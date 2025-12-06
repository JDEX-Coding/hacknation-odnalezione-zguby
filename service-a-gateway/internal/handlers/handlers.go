package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"math"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/hacknation/odnalezione-zguby/service-a-gateway/internal/models"
	"github.com/hacknation/odnalezione-zguby/service-a-gateway/internal/services"
	"github.com/hacknation/odnalezione-zguby/service-a-gateway/internal/storage"
	"github.com/rs/zerolog/log"
)

// Handler contains all HTTP handlers
type Handler struct {
	templates     map[string]*template.Template // Map of page name -> template set
	storage       *storage.MinIOStorage
	rabbitMQ      *services.RabbitMQPublisher
	visionService *services.VisionService
	items         *storage.PostgresStorage // Persistent Postgres storage
}

// NewHandler creates a new handler instance
func NewHandler(
	templatesPath string,
	storage *storage.MinIOStorage,
	rabbitMQ *services.RabbitMQPublisher,
	visionService *services.VisionService,
	itemStorage *storage.PostgresStorage,
) (*Handler, error) {
	// Function map for templates
	funcMap := template.FuncMap{
		"timeRemaining": func(foundDate time.Time, category string) string {
			// Documents have special procedure (police/issuer), no 2-year ownership transfer
			if strings.ToLower(category) == "dokumenty" {
				return "Procedura specjalna"
			}

			// Legislation: 2 years storage for unknown owner (Art. 187 KC)
			expirationDate := foundDate.AddDate(2, 0, 0)
			remaining := time.Until(expirationDate)

			if remaining <= 0 {
				return "Czas minął"
			}

			if remaining.Hours() < 24 {
				return "Mniej niż 24h"
			}

			days := int(remaining.Hours() / 24)
			if days < 30 {
				return fmt.Sprintf("%d dni", days)
			}

			months := int(math.Floor(remaining.Hours() / 24 / 30))
			if months < 12 {
				return fmt.Sprintf("%d mies.", months)
			}

			years := int(math.Floor(remaining.Hours() / 24 / 365))
			monthsRemaining := months % 12
			if monthsRemaining > 0 {
				return fmt.Sprintf("%d rok %d mies.", years, monthsRemaining)
			}
			return fmt.Sprintf("%d rok", years)
		},
		"isExpiringSoon": func(foundDate time.Time, category string) bool {
			if strings.ToLower(category) == "dokumenty" {
				return false // Don't show red warning for documents, just special status
			}
			expirationDate := foundDate.AddDate(2, 0, 0)
			remaining := time.Until(expirationDate)
			// Warning if less than 3 months
			return remaining.Hours() < 24*90
		},
		"isDocuments": func(category string) bool {
			return strings.ToLower(category) == "dokumenty"
		},
		"expirationDate": func(foundDate time.Time, category string) string {
			if strings.ToLower(category) == "dokumenty" {
				return "-"
			}
			// Legislation: 2 years storage
			exp := foundDate.AddDate(2, 0, 0)
			return exp.Format("02.01.2006")
		},
		"daysRemaining": func(foundDate time.Time, category string) int {
			if strings.ToLower(category) == "dokumenty" {
				return 0
			}
			exp := foundDate.AddDate(2, 0, 0)
			remaining := time.Until(exp)
			if remaining <= 0 {
				return 0
			}
			// Round up to nearest day
			days := int(math.Ceil(remaining.Hours() / 24))
			return days
		},
	}

	// Parse base template
	basePath := filepath.Join(templatesPath, "base.html")
	baseTmpl, err := template.New("base.html").Funcs(funcMap).ParseFiles(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base template: %w", err)
	}

	// Helper to create page templates
	pages := []string{"index.html", "create.html", "browse.html", "detail.html"}
	templates := make(map[string]*template.Template)

	for _, page := range pages {
		// Clone base template
		tmpl, err := baseTmpl.Clone()
		if err != nil {
			return nil, fmt.Errorf("failed to clone base template for %s: %w", page, err)
		}

		// Parse page template
		pagePath := filepath.Join(templatesPath, page)
		_, err = tmpl.ParseFiles(pagePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse page template %s: %w", page, err)
		}

		templates[page] = tmpl
	}

	return &Handler{
		templates:     templates,
		storage:       storage,
		rabbitMQ:      rabbitMQ,
		visionService: visionService,
		items:         itemStorage,
	}, nil
}

// HomeHandler handles the home page
func (h *Handler) HomeHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Odnalezione Zguby - System zgłaszania rzeczy znalezionych",
	}

	// Render the index page using the base template
	tmpl, ok := h.templates["index.html"]
	if !ok {
		log.Error().Msg("Template index.html not found")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		log.Error().Err(err).Msg("Failed to render home template")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// CreateFormHandler shows the form to create a new lost item
func (h *Handler) CreateFormHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":      "Dodaj Zgubę",
		"Categories": models.Categories,
	}

	// Render the create page using the base template
	tmpl, ok := h.templates["create.html"]
	if !ok {
		log.Error().Msg("Template create.html not found")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		log.Error().Err(err).Msg("Failed to render create template")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// CreateHandler handles creating a new lost item
func (h *Handler) CreateHandler(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse form")
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Extract form data
	title := r.FormValue("title")
	description := r.FormValue("description")
	category := r.FormValue("category")
	location := r.FormValue("location")
	foundDateStr := r.FormValue("found_date")
	reportingDateStr := r.FormValue("reporting_date")
	contactEmail := r.FormValue("contact_email")
	contactPhone := r.FormValue("contact_phone")
	reportingLocation := r.FormValue("reporting_location")
	imageURL := r.FormValue("imageUrl") // URL from AI analysis (already in MinIO)

	// Validate required fields
	if title == "" || description == "" || location == "" || foundDateStr == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Parse found date
	foundDate, err := time.Parse("2006-01-02", foundDateStr)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse found date")
		http.Error(w, "Invalid found date format", http.StatusBadRequest)
		return
	}

	// Parse reporting date
	var reportingDate time.Time
	if reportingDateStr != "" {
		reportingDate, err = time.Parse("2006-01-02", reportingDateStr)
		if err != nil {
			log.Error().Err(err).Msg("Failed to parse reporting date")
			http.Error(w, "Invalid reporting date format", http.StatusBadRequest)
			return
		}
	} else {
		reportingDate = time.Now()
	}

	// If imageUrl is not provided from AI analysis, try to get it from form file upload
	if imageURL == "" {
		file, header, err := r.FormFile("image")
		if err != nil {
			log.Error().Err(err).Msg("Failed to get file from form")
			http.Error(w, "Image is required", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Validate file type
		contentType := header.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "image/") {
			http.Error(w, "Only image files are allowed", http.StatusBadRequest)
			return
		}

		// Upload to MinIO
		ctx := r.Context()
		imageURL, err = h.storage.UploadImage(ctx, file, header.Filename, contentType, header.Size)
		if err != nil {
			log.Error().Err(err).Msg("Failed to upload image")
			http.Error(w, "Failed to upload image", http.StatusInternalServerError)
			return
		}
	}

	// Create lost item
	itemID := uuid.New().String()
	now := time.Now()

	item := &models.LostItem{
		ID:                itemID,
		Title:             title,
		Description:       description,
		Category:          category,
		Location:          location,
		FoundDate:         foundDate,
		ReportingDate:     reportingDate,
		ReportingLocation: reportingLocation,
		ImageURL:          imageURL,
		Status:            "pending",
		ContactEmail:      contactEmail,
		ContactPhone:      contactPhone,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	// Store in persistent DB
	if err := h.items.Save(item); err != nil {
		log.Error().Err(err).Msg("Failed to save item to DB")
		http.Error(w, "Failed to save item", http.StatusInternalServerError)
		return
	}

	// Publish event to RabbitMQ
	event := models.ItemSubmittedEvent{
		ID:                item.ID,
		Title:             item.Title,
		Description:       item.Description,
		Category:          item.Category,
		Location:          item.Location,
		FoundDate:         item.FoundDate,
		ReportingDate:     item.ReportingDate,
		ReportingLocation: item.ReportingLocation,
		ImageURL:          item.ImageURL,
		ContactEmail:      item.ContactEmail,
		ContactPhone:      item.ContactPhone,
		Timestamp:         now,
	}

	if err := h.rabbitMQ.PublishItemSubmitted(r.Context(), event); err != nil {
		log.Error().Err(err).Msg("Failed to publish event to RabbitMQ")
		// Don't fail the request - item is still created
	}

	log.Info().
		Str("item_id", itemID).
		Str("title", title).
		Msg("Lost item created successfully")

	// Return success response (HTMX will handle this)
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("HX-Redirect", fmt.Sprintf("/zguba/%s", itemID))
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `<div class="alert alert-success">Zguba została dodana pomyślnie!</div>`)
}

// BrowseHandler shows list of all lost items
func (h *Handler) BrowseHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query params
	query := strings.ToLower(r.URL.Query().Get("search"))
	category := strings.ToLower(r.URL.Query().Get("category"))
	status := strings.ToLower(r.URL.Query().Get("status"))

	// Filter items
	filteredItems := make([]*models.LostItem, 0)

	allItems, err := h.items.List()
	if err != nil {
		log.Error().Err(err).Msg("Failed to list items")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	for _, item := range allItems {
		// Filter by search query
		if query != "" && !strings.Contains(strings.ToLower(item.Title), query) &&
			!strings.Contains(strings.ToLower(item.Description), query) &&
			!strings.Contains(strings.ToLower(item.Location), query) {
			continue
		}

		// Filter by category
		if category != "" && strings.ToLower(item.Category) != category {
			continue
		}

		// Filter by status
		if status != "" && strings.ToLower(item.Status) != status {
			continue
		}

		filteredItems = append(filteredItems, item)
	}

	data := map[string]interface{}{
		"Title": "Przeglądaj Zguby",
		"Items": filteredItems,
	}

	// Get the browse template
	tmpl, ok := h.templates["browse.html"]
	if !ok {
		log.Error().Msg("Template browse.html not found")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check for HTMX request
	if r.Header.Get("HX-Request") == "true" {
		// Render only the items list partial
		if err := tmpl.ExecuteTemplate(w, "items-list", data); err != nil {
			log.Error().Err(err).Msg("Failed to render items-list template")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Full page render
	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		log.Error().Err(err).Msg("Failed to render browse template")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// ItemDetailHandler shows details of a single lost item
func (h *Handler) ItemDetailHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	itemID := vars["id"]

	item, exists := h.items.Get(itemID)
	if !exists {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	data := map[string]interface{}{
		"Title": item.Title,
		"Item":  item,
	}

	tmpl, ok := h.templates["detail.html"]
	if !ok {
		log.Error().Msg("Template detail.html not found")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		log.Error().Err(err).Msg("Failed to render detail template")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// AnalyzeImageHandler handles image analysis with Vision API
func (h *Handler) AnalyzeImageHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse form")
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get uploaded file
	file, header, err := r.FormFile("image")
	if err != nil {
		log.Error().Err(err).Msg("Failed to get file from form")
		http.Error(w, "Image is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file type
	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		http.Error(w, "Only image files are allowed", http.StatusBadRequest)
		return
	}

	// Read image into memory and encode as base64
	imageBuffer := new(bytes.Buffer)
	_, err = io.Copy(imageBuffer, file)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read image")
		http.Error(w, "Failed to read image", http.StatusInternalServerError)
		return
	}

	imageBytes := imageBuffer.Bytes()
	imageBase64 := base64.StdEncoding.EncodeToString(imageBytes)

	// Analyze with Vision API using base64 (quick analysis)
	ctx := r.Context()
	analysis, err := h.visionService.AnalyzeImageBase64(ctx, imageBase64)
	if err != nil {
		log.Error().Err(err).Msg("Failed to analyze image")
		// Return a default response instead of failing
		analysis = &services.AnalyzeImageResponse{
			Title:       "Znaleziony przedmiot",
			Description: "Nie udało się automatycznie opisać przedmiotu. Proszę wpisać opis ręcznie.",
			Category:    "Inne",
			Confidence:  "low",
		}
	}

	// Upload original image to MinIO for Service B (Python Worker) to process later
	imageReader := bytes.NewReader(imageBytes)
	imageURL, err := h.storage.UploadImage(ctx, imageReader, header.Filename, contentType, int64(len(imageBytes)))
	if err != nil {
		log.Error().Err(err).Msg("Failed to upload image to MinIO")
		// Don't fail the request - analysis was successful
		imageURL = ""
	}

	// Return JSON response with analysis and image URL
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"title":       analysis.Title,
		"description": analysis.Description,
		"category":    analysis.Category,
		"confidence":  analysis.Confidence,
		"imageUrl":    imageURL,
	}
	json.NewEncoder(w).Encode(response)
}

// AnalyzeImageFormHandler returns HTMX partial with AI-suggested description
func (h *Handler) AnalyzeImageFormHandler(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse form")
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get uploaded file
	file, header, err := r.FormFile("image")
	if err != nil {
		log.Error().Err(err).Msg("Failed to get file from form")
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<div class="text-red-600">Błąd: Nie wybrano pliku</div>`)
		return
	}
	defer file.Close()

	// Read file content and encode as base64
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read file")
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<div class="text-red-600">Błąd: Nie można odczytać pliku</div>`)
		return
	}

	imageBase64 := base64.StdEncoding.EncodeToString(fileBytes)

	// Analyze with Vision API using base64
	ctx := r.Context()
	analysis, err := h.visionService.AnalyzeImageBase64(ctx, imageBase64)
	if err != nil {
		log.Error().Err(err).Msg("Failed to analyze image")
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
			<div class="bg-yellow-50 border border-yellow-200 rounded-lg p-4 mt-4">
				<p class="text-yellow-800">⚠️ Nie udało się automatycznie przeanalizować obrazu. Proszę wpisać opis ręcznie.</p>
			</div>
		`)
		return
	}

	// Upload image to MinIO for Service B (Python Worker)
	imageURL, err := h.storage.UploadImage(ctx, bytes.NewReader(fileBytes), header.Filename, "image/jpeg", int64(len(fileBytes)))
	if err != nil {
		log.Error().Err(err).Msg("Failed to upload image to MinIO")
		// Continue - analysis was successful
	}

	// Return HTMX partial with the analysis and store imageURL as data attribute
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
		<div id="ai-suggestion" class="bg-blue-50 border border-blue-200 rounded-lg p-4 mt-4" data-image-url="%s">
			<h3 class="font-semibold text-blue-900 mb-2">✨ Sugestia AI:</h3>
			<p class="text-blue-800 mb-3">%s</p>
			<div class="flex items-center gap-2 text-sm text-blue-700">
				<span class="font-medium">Kategoria:</span>
				<span class="px-2 py-1 bg-blue-100 rounded">%s</span>
				<span class="ml-2 text-xs">Pewność: %s</span>
			</div>
			<button
				type="button"
				onclick="document.getElementById('description').value = '%s'; document.getElementById('category').value = '%s'; document.getElementById('imageUrl').value = '%s';"
				class="mt-3 px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
			>
				Użyj tej sugestii
			</button>
		</div>
	`, imageURL, analysis.Description, analysis.Category, analysis.Confidence,
		strings.ReplaceAll(analysis.Description, "'", "\\'"), analysis.Category, imageURL)
}

// SearchRequest represents the search form
type SearchRequest struct {
	Query string `json:"query"`
}

type ClipEmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

type QdrantSearchRequest struct {
	Embedding []float32 `json:"embedding"`
	Limit     int       `json:"limit"`
}

type QdrantSearchResult struct {
	ID      string          `json:"id"`
	Score   float32         `json:"score"`
	Payload models.LostItem `json:"payload"`
}

// SemanticSearchHandler handles semantic search requests
func (h *Handler) SemanticSearchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("query")
	if query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	// 1. Get Embedding from Clip Service
	// Assuming Clip Service is at http://clip-service:8000
	clipResp, err := http.Post(
		"http://clip-service:8000/embed",
		"application/json",
		strings.NewReader(fmt.Sprintf(`{"text": "%s"}`, query)),
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to call Clip Service")
		http.Error(w, "Search service unavailable (Clip)", http.StatusServiceUnavailable)
		return
	}
	defer clipResp.Body.Close()

	if clipResp.StatusCode != http.StatusOK {
		log.Error().Int("status", clipResp.StatusCode).Msg("Clip Service returned error")
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	var embeddingResp ClipEmbeddingResponse
	if err := json.NewDecoder(clipResp.Body).Decode(&embeddingResp); err != nil {
		log.Error().Err(err).Msg("Failed to decode embedding response")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 2. Search Qdrant
	// Assuming Qdrant Service is at http://qdrant-service:8081
	qdrantReqBody, _ := json.Marshal(QdrantSearchRequest{
		Embedding: embeddingResp.Embedding,
		Limit:     20,
	})

	qdrantResp, err := http.Post(
		"http://qdrant-service:8081/search",
		"application/json",
		bytes.NewReader(qdrantReqBody),
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to call Qdrant Service")
		http.Error(w, "Search service unavailable (Qdrant)", http.StatusServiceUnavailable)
		return
	}
	defer qdrantResp.Body.Close()

	var searchResults []QdrantSearchResult
	if err := json.NewDecoder(qdrantResp.Body).Decode(&searchResults); err != nil {
		log.Error().Err(err).Msg("Failed to decode search results")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 3. Render Results
	// Map results to LostItem model
	var items []*models.LostItem
	for _, res := range searchResults {
		// Use payload from Qdrant directly, or better: fetch from Postgres to be sure?
		// Payload in Qdrant might be partial. But for now it's fine.
		// Actually, let's fetch from DB using ID to get full consistent state
		if item, exists := h.items.Get(res.ID); exists {
			items = append(items, item)
		}
	}

	data := map[string]interface{}{
		"Title": "Wyniki wyszukiwania: " + query,
		"Items": items,
		"Query": query,
	}

	tmpl, ok := h.templates["browse.html"]
	if !ok {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		tmpl.ExecuteTemplate(w, "items-list", data)
		return
	}

	tmpl.ExecuteTemplate(w, "base.html", data)
}

// HealthCheckHandler returns health status
func (h *Handler) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// ... existing content ...
	ctx := r.Context()

	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]string{
			"storage":  "ok",
			"rabbitmq": "ok",
			"vision":   "ok",
		},
	}

	// Check MinIO
	if err := h.storage.HealthCheck(ctx); err != nil {
		health["status"] = "unhealthy"
		health["checks"].(map[string]string)["storage"] = err.Error()
	}

	// Check RabbitMQ
	if err := h.rabbitMQ.HealthCheck(); err != nil {
		health["status"] = "unhealthy"
		health["checks"].(map[string]string)["rabbitmq"] = err.Error()
	}

	// Check Vision API
	if err := h.visionService.HealthCheck(ctx); err != nil {
		health["status"] = "unhealthy"
		health["checks"].(map[string]string)["vision"] = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	statusCode := http.StatusOK
	if health["status"] == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(health)
}
