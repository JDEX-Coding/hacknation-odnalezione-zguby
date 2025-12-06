package models

import "time"

// ItemVectorizedEvent represents the event consumed from RabbitMQ
type ItemVectorizedEvent struct {
	ID                string    `json:"id"`
	Title             string    `json:"title"`
	Description       string    `json:"description"`
	Category          string    `json:"category"`
	Location          string    `json:"location"`
	FoundDate         time.Time `json:"found_date"`
	ReportingDate     time.Time `json:"reporting_date"`
	ReportingLocation string    `json:"reporting_location"`
	ImageURL          string    `json:"image_url"`
	ContactEmail      string    `json:"contact_email"`
	ContactPhone      string    `json:"contact_phone"`
	VectorID          string    `json:"vector_id,omitempty"`
	Timestamp         time.Time `json:"timestamp"`
}

// ItemPublishedEvent represents successful publication event
type ItemPublishedEvent struct {
	ID              string    `json:"id"`
	DatasetID       string    `json:"dataset_id"`
	PublishedAt     time.Time `json:"published_at"`
	DaneGovURL      string    `json:"dane_gov_url,omitempty"`
	PublicationDate time.Time `json:"publication_date"`
}
