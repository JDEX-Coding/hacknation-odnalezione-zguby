# Database Schema

## Tables

### `datasets`

Stores information about datasets that group multiple lost items together.

| Column            | Type         | Constraints          | Description                                      |
|-------------------|--------------|----------------------|--------------------------------------------------|
| id                | VARCHAR(36)  | PRIMARY KEY          | Unique dataset identifier (UUID)                 |
| title             | TEXT         | NOT NULL             | Dataset title (e.g., "Rejestr rzeczy znalezionych 2025") |
| notes             | TEXT         |                      | Dataset description/notes                        |
| url               | TEXT         |                      | URL to JSON representation of all items in dataset |
| institution_name  | TEXT         | NOT NULL             | Institution/organization name                    |
| email             | TEXT         | NOT NULL             | Contact email                                    |
| categories        | TEXT[]       |                      | Array of categories (e.g., ["transport", "inne"]) |
| tags              | TEXT[]       |                      | Array of tags (e.g., ["rzeczy znalezione", "2025"]) |
| created_at        | TIMESTAMP    | NOT NULL, DEFAULT NOW() | Creation timestamp                            |
| updated_at        | TIMESTAMP    | NOT NULL, DEFAULT NOW() | Last update timestamp                         |

### `lost_items`

Stores individual lost item records (existing table).

| Column                | Type         | Constraints          | Description                          |
|-----------------------|--------------|----------------------|--------------------------------------|
| id                    | VARCHAR(36)  | PRIMARY KEY          | Unique item identifier (UUID)        |
| title                 | TEXT         | NOT NULL             | Item title                           |
| description           | TEXT         |                      | Item description                     |
| category              | VARCHAR(100) |                      | Item category                        |
| location              | TEXT         |                      | Where item was found                 |
| found_date            | TIMESTAMP    |                      | Date when item was found             |
| reporting_date        | TIMESTAMP    |                      | Date when item was reported          |
| reporting_location    | TEXT         |                      | Where item was reported              |
| image_url             | TEXT         |                      | URL to item image                    |
| image_key             | TEXT         |                      | MinIO storage key for image          |
| status                | VARCHAR(50)  |                      | Item status (pending/published/archived) |
| contact_email         | TEXT         |                      | Contact email                        |
| contact_phone         | TEXT         |                      | Contact phone                        |
| processed_by_clip     | BOOLEAN      | DEFAULT FALSE        | CLIP processing status               |
| processed_by_qdrant   | BOOLEAN      | DEFAULT FALSE        | Qdrant vectorization status          |
| published_on_dane_gov | BOOLEAN      | DEFAULT FALSE        | dane.gov.pl publication status       |
| created_at            | TIMESTAMP    |                      | Creation timestamp                   |
| updated_at            | TIMESTAMP    |                      | Last update timestamp                |

### `dataset_items`

Junction table linking datasets to lost items (many-to-many relationship).

| Column      | Type         | Constraints                                    | Description                      |
|-------------|--------------|------------------------------------------------|----------------------------------|
| dataset_id  | VARCHAR(36)  | NOT NULL, FOREIGN KEY → datasets(id) CASCADE   | Reference to dataset             |
| item_id     | VARCHAR(36)  | NOT NULL, FOREIGN KEY → lost_items(id) CASCADE | Reference to lost item           |
| added_at    | TIMESTAMP    | NOT NULL, DEFAULT NOW()                        | When item was added to dataset   |
|             |              | PRIMARY KEY (dataset_id, item_id)              | Composite primary key            |

**Indexes:**
- `idx_dataset_items_dataset_id` on `dataset_id`
- `idx_dataset_items_item_id` on `item_id`

## Relationships

```
datasets (1) ──────< dataset_items >────── (1) lost_items
                    (many-to-many)
```

A dataset can contain multiple lost items, and a lost item can belong to multiple datasets.

## JSON Schema for Publisher API

When the publisher sends dataset information to dane.gov.pl, it uses the following structure:

```json
{
  "title": "Rejestr rzeczy znalezionych 2025",
  "notes": "Baza danych zgłoszonych rzeczy znalezionych w 2025 roku",
  "url": "https://example.com/dataset-url.json",
  "institution_name": "Urząd Miasta",
  "email": "kontakt@urzad.pl",
  "categories": ["transport", "inne"],
  "tags": ["rzeczy znalezione", "2025"]
}
```

The `url` field should point to a JSON endpoint that returns all lost items associated with the dataset (retrieved via the `dataset_items` junction table).

## Available Storage Methods

### Dataset Operations

- `SaveDataset(dataset *models.Dataset) error` - Create or update a dataset
- `GetDataset(id string) (*models.Dataset, bool)` - Retrieve a dataset by ID
- `ListDatasets() ([]*models.Dataset, error)` - List all datasets
- `GetDatasetWithItems(datasetID string) (*models.DatasetWithItems, error)` - Get dataset with all its items

### Dataset-Item Association Operations

- `AddItemToDataset(datasetID, itemID string) error` - Add an item to a dataset
- `RemoveItemFromDataset(datasetID, itemID string) error` - Remove an item from a dataset

### Lost Item Operations (existing)

- `Save(item *models.LostItem) error` - Create or update a lost item
- `Get(id string) (*models.LostItem, bool)` - Retrieve an item by ID
- `List() ([]*models.LostItem, error)` - List all items

## Go Models

### Dataset Model

```go
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
```

### DatasetWithItems Model

```go
type DatasetWithItems struct {
    Dataset
    Items []*LostItem `json:"items"`
}
```

## Migration Notes

When you start the `a-gateway` service, the `Init()` method will automatically:

1. Create the `datasets` table if it doesn't exist
2. Create the `dataset_items` junction table if it doesn't exist
3. Create indexes on the junction table for optimal query performance

No manual migration is required - the schema will be created/updated automatically on service startup.
