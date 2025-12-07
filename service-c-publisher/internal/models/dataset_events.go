package models

import "time"

// DatasetPublishEvent represents an event to publish a dataset to dane.gov.pl
type DatasetPublishEvent struct {
	DatasetID       string    `json:"dataset_id"`
	Title           string    `json:"title"`
	Notes           string    `json:"notes"`
	URL             string    `json:"url"`
	InstitutionName string    `json:"institution_name"`
	Email           string    `json:"email"`
	Categories      []string  `json:"categories"`
	Tags            []string  `json:"tags"`
	Timestamp       time.Time `json:"timestamp"`
}

// DatasetPublishedEvent represents successful dataset publication
type DatasetPublishedEvent struct {
	DatasetID       string    `json:"dataset_id"`
	DaneGovID       string    `json:"dane_gov_id"`
	PublishedAt     time.Time `json:"published_at"`
	DaneGovURL      string    `json:"dane_gov_url,omitempty"`
	PublicationDate time.Time `json:"publication_date"`
}
