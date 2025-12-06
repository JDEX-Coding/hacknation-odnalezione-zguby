package models

import (
	"time"
)

// LostItem represents a lost item entry
type LostItem struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Location    string    `json:"location"`
	FoundDate   time.Time `json:"found_date"`
	ImageURL    string    `json:"image_url"`
	Status      string    `json:"status"` // pending, published, archived
	ContactInfo string    `json:"contact_info"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateLostItemRequest represents the form data for creating a lost item
type CreateLostItemRequest struct {
	Title       string `json:"title" form:"title"`
	Description string `json:"description" form:"description"`
	Category    string `json:"category" form:"category"`
	Location    string `json:"location" form:"location"`
	FoundDate   string `json:"found_date" form:"found_date"`
	ContactInfo string `json:"contact_info" form:"contact_info"`
}

// ItemSubmittedEvent represents the event published to RabbitMQ
type ItemSubmittedEvent struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Location    string    `json:"location"`
	FoundDate   time.Time `json:"found_date"`
	ImageURL    string    `json:"image_url"`
	ContactInfo string    `json:"contact_info"`
	Timestamp   time.Time `json:"timestamp"`
}

// Category options
var Categories = []string{
	"Dokumenty",
	"Elektronika",
	"Biżuteria",
	"Odzież",
	"Portfele i torby",
	"Klucze",
	"Telefony",
	"Zwierzęta",
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
