## Ingestion Service
// Endpoint: POST /lost-items
// Content-Type: multipart/form-data
```
type LostItemRequest struct {
    // Plik jest obsługiwany osobno jako multipart file ("image")
    UserDescription string  `json:"user_description" form:"user_description" binding:"required"`
    LocationLabel   string  `json:"location_label" form:"location_label"` // np. "Park Saski"
    Latitude        float64 `json:"latitude" form:"latitude" binding:"required"`
    Longitude       float64 `json:"longitude" form:"longitude" binding:"required"`
    UserContact     string  `json:"user_contact" form:"user_contact"` // Email/Telefon (szyfrowane w bazie)
    CategoryGuess   string  `json:"category_guess" form:"category_guess"`
}

```

### Internal Event Payload
```
type ItemIngestedEvent struct {
    RequestID     string    `json:"request_id"`      // UUID
    Timestamp     time.Time `json:"timestamp"`
    MinioBucket   string    `json:"minio_bucket"`    // "lost-items-raw"
    MinioPath     string    `json:"minio_path"`      // "2024/05/uuid-original.jpg"
    UserDesc      string    `json:"user_desc"`
    GeoLocation   GeoPoint  `json:"geo_location"`
}
```

```
type GeoPoint struct {
    Lat float64 `json:"lat"`
    Lon float64 `json:"lon"`
}
```

## AI Processor Service

```
type VisionAnalysisResult struct {
    Category      string   `json:"category"`
    Color         string   `json:"color"`
    Brand         string   `json:"brand,omitempty"`
    Features      []string `json:"distinctive_features"`
    Description   string   `json:"visual_description_for_embedding"`
    ContainsPII   bool     `json:"contains_pii"` // Co z danymi osobowymi / RODO
    IsSafeToPub   bool     `json:"is_safe_to_publish"`
}
```

### Internal Event Payload
```
type ItemProcessedEvent struct {
    RequestID      string    `json:"request_id"`
    ProcessingTime int64     `json:"processing_time_ms"`
    Analysis       VisionAnalysisResult `json:"analysis"`
    VectorEmbedding []float32 `json:"vector_embedding"`
    RawImagePath    string `json:"raw_image_path"`
    PublicImageURL  string `json:"public_image_url"`
}
```

## Gov Integration Payload (Export do dane.gov.pl)

// Struktura zgodna z DCAT-AP PL (uproszczona do JSON)
```
type DaneGovPlRecord struct {
    IdEwidencyjny     string `json:"id_ewidencyjny"`      // Nasz UUID
    NazwaPrzedmiotu   string `json:"nazwa_przedmiotu"`    // np. "Telefon Samsung"
    Kategoria         string `json:"kategoria"`           // "Elektronika"
    DataZnalezienia   string `json:"data_znalezienia"`    // Format "YYYY-MM-DD"
    MiejsceGmina      string `json:"miejsce_gmina"`       // np. "Warszawa"
    MiejsceOpis       string `json:"miejsce_opis"`        // "Park Saski" (bez dokładnych koordynatów!)
    CechySzczegolne   string `json:"cechy_szczegolne"`    // "Rysa na ekranie"
    JednostkaZglaszajaca string `json:"jednostka_zglaszajaca"` // "System Zgub v1"
    LinkDoZdjecia     string `json:"link_do_zdjecia"`     // URL z Public Bucket
    Status            string `json:"status"`              // "Do odbioru"
}

// Wrapper dla API dane.gov.pl (jeśli wysyłasz jako JSON-API)
type GovAPIRequest struct {
    Data struct {
        Type       string            `json:"type"` // "resource"
        Attributes map[string]string `json:"attributes"` // Tutaj link do pliku JSON lub opis
    } `json:"data"`
}
```
