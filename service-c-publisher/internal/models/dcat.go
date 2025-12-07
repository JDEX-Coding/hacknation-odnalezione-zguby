package models

import "time"

// DCATDataset represents a dataset in DCAT-AP PL format
type DCATDataset struct {
	Context      string             `json:"@context"`
	Type         string             `json:"@type"`
	ID           string             `json:"@id"`
	Title        DCATLangString     `json:"dct:title"`
	Description  DCATLangString     `json:"dct:description"`
	Issued       string             `json:"dct:issued"`
	Modified     string             `json:"dct:modified"`
	Publisher    DCATPublisher      `json:"dct:publisher"`
	ContactPoint DCATContactPoint   `json:"dcat:contactPoint"`
	Theme        []string           `json:"dcat:theme"`
	Keyword      []string           `json:"dcat:keyword"`
	Spatial      *DCATSpatial       `json:"dct:spatial,omitempty"`
	Temporal     *DCATTemporal      `json:"dct:temporal,omitempty"`
	Distribution []DCATDistribution `json:"dcat:distribution"`
	License      string             `json:"dct:license"`
	Language     []string           `json:"dct:language"`
}

// DCATLangString represents a language-tagged string
type DCATLangString struct {
	PL string `json:"pl"`
	EN string `json:"en,omitempty"`
}

// DCATPublisher represents the publisher organization
type DCATPublisher struct {
	Type string `json:"@type"`
	Name string `json:"foaf:name"`
}

// DCATContactPoint represents contact information
type DCATContactPoint struct {
	Type  string `json:"@type"`
	Email string `json:"vcard:hasEmail,omitempty"`
	Tel   string `json:"vcard:hasTelephone,omitempty"`
}

// DCATSpatial represents geographic coverage
type DCATSpatial struct {
	Type  string `json:"@type"`
	Label string `json:"rdfs:label"`
}

// DCATTemporal represents temporal coverage
type DCATTemporal struct {
	Type      string `json:"@type"`
	StartDate string `json:"dcat:startDate"`
	EndDate   string `json:"dcat:endDate,omitempty"`
}

// DCATDistribution represents a distribution (resource)
type DCATDistribution struct {
	Type        string         `json:"@type"`
	Title       DCATLangString `json:"dct:title"`
	Description DCATLangString `json:"dct:description,omitempty"`
	Format      string         `json:"dct:format"`
	AccessURL   string         `json:"dcat:accessURL"`
	DownloadURL string         `json:"dcat:downloadURL,omitempty"`
	ByteSize    int64          `json:"dcat:byteSize,omitempty"`
	MediaType   string         `json:"dcat:mediaType,omitempty"`
}

// DatasetRequest represents the request to dane.gov.pl API
type DatasetRequest struct {
	Data DatasetData `json:"data"`
}

// DatasetData wraps the dataset attributes
type DatasetData struct {
	Type       string            `json:"type"`
	Attributes DatasetAttributes `json:"attributes"`
}

// DatasetAttributes contains the dataset metadata
type DatasetAttributes struct {
	Title           string              `json:"title"`
	Notes           string              `json:"notes"`
	Category        string              `json:"category"`
	Status          string              `json:"status"`
	Visibility      string              `json:"visibility"`
	UpdateFrequency string              `json:"update_frequency"`
	Tags            []string            `json:"tags"`
	License         string              `json:"license_id"`
	CustomFields    map[string]string   `json:"custom_fields,omitempty"`
	Resources       []ResourceAttribute `json:"resources,omitempty"`
	OrganizationID  string              `json:"organization_id,omitempty"`
	PublicationDate time.Time           `json:"publication_date"`
}

// ResourceAttribute represents a resource/file attached to the dataset
type ResourceAttribute struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Format      string `json:"format"`
	URL         string `json:"url"`
	Size        int64  `json:"size,omitempty"`
}

// DatasetResponse represents the API response
type DatasetResponse struct {
	Data struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			Title  string    `json:"title"`
			Slug   string    `json:"slug"`
			Status string    `json:"status"`
			URL    string    `json:"url"`
			Date   time.Time `json:"created"`
		} `json:"attributes"`
	} `json:"data"`
	Links struct {
		Self string `json:"self"`
	} `json:"links"`
}

// ResourceRequest represents the request to add a resource to a dataset
type ResourceRequest struct {
	Data ResourceData `json:"data"`
}

// ResourceData wraps the resource attributes
type ResourceData struct {
	Type       string                  `json:"type"`
	Attributes ResourceAttributeDetail `json:"attributes"`
}

// ResourceAttributeDetail contains the resource metadata
type ResourceAttributeDetail struct {
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	Format       string            `json:"format"`
	URL          string            `json:"url"`
	Size         int64             `json:"size,omitempty"`
	CustomFields map[string]string `json:"custom_fields,omitempty"`
}

// ResourceResponse represents the API response for resource creation
type ResourceResponse struct {
	Data struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			Name   string    `json:"name"`
			Format string    `json:"format"`
			URL    string    `json:"url"`
			Date   time.Time `json:"created"`
		} `json:"attributes"`
	} `json:"data"`
	Links struct {
		Self string `json:"self"`
	} `json:"links"`
}
