package models

import (
	"time"
)

// LostItem represents a lost item entry
type LostItem struct {
	ID                 string    `json:"id"`
	Title              string    `json:"title"`
	Description        string    `json:"description"`
	Category           string    `json:"category"`
	Location           string    `json:"location"`
	FoundDate          time.Time `json:"found_date"`
	ReportingDate      time.Time `json:"reporting_date"`
	ReportingLocation  string    `json:"reporting_location"`
	ImageURL           string    `json:"image_url"`
	ImageKey           string    `json:"image_key"`
	Status             string    `json:"status"` // pending, published, archived
	ProcessedByClip    bool      `json:"processed_by_clip"`
	ProcessedByQdrant  bool      `json:"processed_by_qdrant"`
	PublishedOnDaneGov bool      `json:"published_on_dane_gov"`
	ContactEmail       string    `json:"contact_email"`
	ContactPhone       string    `json:"contact_phone"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
type CreateLostItemRequest struct {
	Title        string `json:"title" form:"title"`
	Description  string `json:"description" form:"description"`
	Category     string `json:"category" form:"category"`
	Location     string `json:"location" form:"location"`
	FoundDate    string `json:"found_date" form:"found_date"`
	ContactEmail string `json:"contact_email" form:"contact_email"`
	ContactPhone string `json:"contact_phone" form:"contact_phone"`
}

// ItemSubmittedEvent represents the event published to RabbitMQ
type ItemSubmittedEvent struct {
	ID                string    `json:"item_id"`
	Title             string    `json:"text"`
	Description       string    `json:"description"`
	Category          string    `json:"category"`
	Location          string    `json:"location"`
	FoundDate         time.Time `json:"date_lost"`
	ReportingDate     time.Time `json:"reporting_date"`
	ReportingLocation string    `json:"reporting_location"`
	ImageURL          string    `json:"image_url"`
	ImageKey          string    `json:"image_key"`
	ContactEmail      string    `json:"contact_email"`
	ContactPhone      string    `json:"contact_phone"`
	Timestamp         time.Time `json:"timestamp"`
}

var Categories = []string{
	"Dokumenty",
	"Elektronika",
	"Biżuteria",
	"Odzież",
	"Portfele i torby",
	"Klucze",
	"Telefony",
	"Inne",
}

// VisionAnalysisRequest represents request to Vision API
type VisionAnalysisRequest struct {
	ImageBase64 string `json:"image_base64"`
}

// VisionAnalysisResponse represents response from Vision API
type VisionAnalysisResponse struct {
	Description string `json:"description"`
	Category    string `json:"category"`
	Error       string `json:"error,omitempty"`
}
