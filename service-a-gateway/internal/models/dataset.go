package models

import (
	"time"
)

// Dataset represents a dataset containing multiple lost items
type Dataset struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Notes           string    `json:"notes"`
	URL             string    `json:"url"`
	InstitutionName string    `json:"institution_name"`
	Email           string    `json:"email"`
	Categories      []string  `json:"categories"`
	Tags            []string  `json:"tags"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// DatasetWithItems represents a dataset with its associated lost items
type DatasetWithItems struct {
	Dataset
	Items []*LostItem `json:"items"`
}

// CreateDatasetRequest represents the request to create a dataset
type CreateDatasetRequest struct {
	Title           string   `json:"title"`
	Notes           string   `json:"notes"`
	URL             string   `json:"url"`
	InstitutionName string   `json:"institution_name"`
	Email           string   `json:"email"`
	Categories      []string `json:"categories"`
	Tags            []string `json:"tags"`
}
