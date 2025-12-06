package formatter

import (
	"fmt"
	"strings"
	"time"

	"github.com/hacknation/odnalezione-zguby/service-c-publisher/internal/models"
)

// DCATFormatter converts lost items to DCAT-AP PL format
type DCATFormatter struct {
	publisherName string
	publisherID   string
	baseURL       string
}

// NewDCATFormatter creates a new DCAT formatter
func NewDCATFormatter(publisherName, publisherID, baseURL string) *DCATFormatter {
	return &DCATFormatter{
		publisherName: publisherName,
		publisherID:   publisherID,
		baseURL:       baseURL,
	}
}

// FormatToDCAT converts an item to DCAT-AP format
func (f *DCATFormatter) FormatToDCAT(item *models.ItemVectorizedEvent) *models.DCATDataset {
	now := time.Now()
	datasetID := fmt.Sprintf("%s/datasets/%s", f.baseURL, item.ID)

	return &models.DCATDataset{
		Context: "https://www.w3.org/ns/dcat",
		Type:    "dcat:Dataset",
		ID:      datasetID,
		Title: models.DCATLangString{
			PL: item.Title,
			EN: translateTitle(item.Title),
		},
		Description: models.DCATLangString{
			PL: item.Description,
			EN: translateDescription(item.Description),
		},
		Issued:   item.ReportingDate.Format(time.RFC3339),
		Modified: now.Format(time.RFC3339),
		Publisher: models.DCATPublisher{
			Type: "foaf:Organization",
			Name: f.publisherName,
		},
		ContactPoint: models.DCATContactPoint{
			Type:  "vcard:Organization",
			Email: item.ContactEmail,
			Tel:   item.ContactPhone,
		},
		Theme:    f.getThemes(item.Category),
		Keyword:  f.getKeywords(item),
		Spatial:  f.getSpatial(item.Location),
		Temporal: f.getTemporal(item.FoundDate),
		Distribution: []models.DCATDistribution{
			{
				Type: "dcat:Distribution",
				Title: models.DCATLangString{
					PL: "Zdjęcie rzeczy znalezionej",
					EN: "Lost item image",
				},
				Description: models.DCATLangString{
					PL: "Fotografia znalezionego przedmiotu",
					EN: "Photograph of the found item",
				},
				Format:      "image/jpeg",
				AccessURL:   item.ImageURL,
				DownloadURL: item.ImageURL,
				MediaType:   "image/jpeg",
			},
		},
		License:  "https://creativecommons.org/publicdomain/zero/1.0/",
		Language: []string{"pl"},
	}
}

// FormatToDatasetRequest converts to dane.gov.pl API format
func (f *DCATFormatter) FormatToDatasetRequest(item *models.ItemVectorizedEvent) *models.DatasetRequest {
	return &models.DatasetRequest{
		Data: models.DatasetData{
			Type: "dataset",
			Attributes: models.DatasetAttributes{
				Title:           item.Title,
				Notes:           item.Description,
				Category:        f.mapCategory(item.Category),
				Status:          "published",
				Visibility:      "public",
				UpdateFrequency: "onDemand",
				Tags:            f.getKeywords(item),
				License:         "cc-zero",
				CustomFields: map[string]string{
					"found_date":         item.FoundDate.Format("2006-01-02"),
					"location":           item.Location,
					"reporting_location": item.ReportingLocation,
					"contact_email":      item.ContactEmail,
					"contact_phone":      item.ContactPhone,
				},
				Resources: []models.ResourceAttribute{
					{
						Name:        fmt.Sprintf("Zdjęcie: %s", item.Title),
						Description: fmt.Sprintf("Fotografia znalezionego przedmiotu - %s", item.Category),
						Format:      "JPG",
						URL:         item.ImageURL,
					},
				},
				OrganizationID:  f.publisherID,
				PublicationDate: time.Now(),
			},
		},
	}
}

// getThemes returns DCAT themes based on category
func (f *DCATFormatter) getThemes(category string) []string {
	// Map categories to EU vocabularies
	themeMap := map[string]string{
		"Dokumenty":        "http://publications.europa.eu/resource/authority/data-theme/JUST",
		"Elektronika":      "http://publications.europa.eu/resource/authority/data-theme/TECH",
		"Biżuteria":        "http://publications.europa.eu/resource/authority/data-theme/ECON",
		"Odzież":           "http://publications.europa.eu/resource/authority/data-theme/SOCI",
		"Portfele i torby": "http://publications.europa.eu/resource/authority/data-theme/ECON",
		"Klucze":           "http://publications.europa.eu/resource/authority/data-theme/TECH",
		"Telefony":         "http://publications.europa.eu/resource/authority/data-theme/TECH",
	}

	if theme, ok := themeMap[category]; ok {
		return []string{theme}
	}
	return []string{"http://publications.europa.eu/resource/authority/data-theme/SOCI"}
}

// getKeywords extracts keywords from item
func (f *DCATFormatter) getKeywords(item *models.ItemVectorizedEvent) []string {
	keywords := []string{
		"rzeczy znalezione",
		"lost and found",
		item.Category,
		strings.ToLower(item.Location),
	}

	// Add words from title
	titleWords := strings.Fields(item.Title)
	for _, word := range titleWords {
		if len(word) > 3 {
			keywords = append(keywords, strings.ToLower(word))
		}
	}

	return keywords
}

// getSpatial returns spatial information
func (f *DCATFormatter) getSpatial(location string) *models.DCATSpatial {
	if location == "" {
		return nil
	}
	return &models.DCATSpatial{
		Type:  "dct:Location",
		Label: location,
	}
}

// getTemporal returns temporal coverage
func (f *DCATFormatter) getTemporal(foundDate time.Time) *models.DCATTemporal {
	return &models.DCATTemporal{
		Type:      "dct:PeriodOfTime",
		StartDate: foundDate.Format("2006-01-02"),
		EndDate:   foundDate.AddDate(0, 0, 90).Format("2006-01-02"), // 90 days retention
	}
}

// mapCategory maps internal categories to dane.gov.pl categories
func (f *DCATFormatter) mapCategory(category string) string {
	categoryMap := map[string]string{
		"Dokumenty":        "government",
		"Elektronika":      "technology",
		"Biżuteria":        "economy",
		"Odzież":           "society",
		"Portfele i torby": "economy",
		"Klucze":           "technology",
		"Telefony":         "technology",
		"Inne":             "other",
	}

	if mapped, ok := categoryMap[category]; ok {
		return mapped
	}
	return "other"
}

// translateTitle provides basic English translation
func translateTitle(title string) string {
	// In production, use a proper translation service
	return fmt.Sprintf("Lost item: %s", title)
}

// translateDescription provides basic English translation
func translateDescription(desc string) string {
	// In production, use a proper translation service
	return fmt.Sprintf("Found item description: %s", desc)
}
