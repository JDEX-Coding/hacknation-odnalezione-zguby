package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/hacknation/odnalezione-zguby/service-a-gateway/internal/models"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(host, port, user, password, dbName, sslMode string) (*PostgresStorage, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbName, sslMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open db connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	storage := &PostgresStorage{db: db}
	if err := storage.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize db schema: %w", err)
	}

	return storage, nil
}

// Init creates necessary tables
func (s *PostgresStorage) Init() error {
	query := `
	CREATE TABLE IF NOT EXISTS lost_items (
		id VARCHAR(36) PRIMARY KEY,
		title TEXT NOT NULL,
		description TEXT,
		category VARCHAR(100),
		location TEXT,
		found_date TIMESTAMP,
		image_url TEXT,
		status VARCHAR(50),
		contact_info TEXT,
		reporting_date TIMESTAMP,
		reporting_location TEXT,
		created_at TIMESTAMP,
		updated_at TIMESTAMP
	);

	ALTER TABLE lost_items ADD COLUMN IF NOT EXISTS reporting_date TIMESTAMP;
	ALTER TABLE lost_items ADD COLUMN IF NOT EXISTS reporting_location TEXT;
	ALTER TABLE lost_items ADD COLUMN IF NOT EXISTS contact_email TEXT;
	ALTER TABLE lost_items ADD COLUMN IF NOT EXISTS contact_phone TEXT;
	ALTER TABLE lost_items ADD COLUMN IF NOT EXISTS processed_by_clip BOOLEAN DEFAULT FALSE;
	ALTER TABLE lost_items ADD COLUMN IF NOT EXISTS processed_by_qdrant BOOLEAN DEFAULT FALSE;
	ALTER TABLE lost_items ADD COLUMN IF NOT EXISTS published_on_dane_gov BOOLEAN DEFAULT FALSE;
	ALTER TABLE lost_items ADD COLUMN IF NOT EXISTS image_key TEXT;

	CREATE TABLE IF NOT EXISTS datasets (
		id VARCHAR(36) PRIMARY KEY,
		title TEXT NOT NULL,
		notes TEXT,
		url TEXT,
		institution_name TEXT NOT NULL,
		email TEXT NOT NULL,
		categories TEXT[],
		tags TEXT[],
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS dataset_items (
		dataset_id VARCHAR(36) NOT NULL REFERENCES datasets(id) ON DELETE CASCADE,
		item_id VARCHAR(36) NOT NULL REFERENCES lost_items(id) ON DELETE CASCADE,
		added_at TIMESTAMP NOT NULL DEFAULT NOW(),
		PRIMARY KEY (dataset_id, item_id)
	);

	CREATE INDEX IF NOT EXISTS idx_dataset_items_dataset_id ON dataset_items(dataset_id);
	CREATE INDEX IF NOT EXISTS idx_dataset_items_item_id ON dataset_items(item_id);`

	_, err := s.db.Exec(query)
	return err
}

func (s *PostgresStorage) Save(item *models.LostItem) error {
	query := `
	INSERT INTO lost_items (
		id, title, description, category, location, found_date,
		reporting_date, reporting_location,
		image_url, image_key, status, contact_email, contact_phone,
		processed_by_clip, processed_by_qdrant, published_on_dane_gov,
		created_at, updated_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
	) ON CONFLICT (id) DO UPDATE SET
		title = EXCLUDED.title,
		description = EXCLUDED.description,
		category = EXCLUDED.category,
		location = EXCLUDED.location,
		found_date = EXCLUDED.found_date,
		reporting_date = EXCLUDED.reporting_date,
		reporting_location = EXCLUDED.reporting_location,
		image_url = EXCLUDED.image_url,
		image_key = EXCLUDED.image_key,
		status = EXCLUDED.status,
		contact_email = EXCLUDED.contact_email,
		contact_phone = EXCLUDED.contact_phone,
		processed_by_clip = EXCLUDED.processed_by_clip,
		processed_by_qdrant = EXCLUDED.processed_by_qdrant,
		published_on_dane_gov = EXCLUDED.published_on_dane_gov,
		updated_at = EXCLUDED.updated_at
	;`

	now := time.Now()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now

	_, err := s.db.Exec(query,
		item.ID, item.Title, item.Description, item.Category, item.Location,
		item.FoundDate, item.ReportingDate, item.ReportingLocation,
		item.ImageURL, item.ImageKey, item.Status, item.ContactEmail, item.ContactPhone,
		item.ProcessedByClip, item.ProcessedByQdrant, item.PublishedOnDaneGov,
		item.CreatedAt, item.UpdatedAt,
	)

	if err != nil {
		log.Error().Err(err).Msg("Failed to save item to postgres")
		return err
	}

	return nil
}

// Get retrieves an item by ID
func (s *PostgresStorage) Get(id string) (*models.LostItem, bool) {
	query := `
	SELECT id, title, description, category, location, found_date,
		   reporting_date, reporting_location,
		   image_url, image_key, status, contact_email, contact_phone,
		   processed_by_clip, processed_by_qdrant, published_on_dane_gov,
		   created_at, updated_at
	FROM lost_items WHERE id = $1`

	item := &models.LostItem{}
	var processedByClip, processedByQdrant, publishedOnDaneGov sql.NullBool
	var imageKey sql.NullString

	err := s.db.QueryRow(query, id).Scan(
		&item.ID, &item.Title, &item.Description, &item.Category, &item.Location,
		&item.FoundDate, &item.ReportingDate, &item.ReportingLocation,
		&item.ImageURL, &imageKey, &item.Status, &item.ContactEmail, &item.ContactPhone,
		&processedByClip, &processedByQdrant, &publishedOnDaneGov,
		&item.CreatedAt, &item.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to get item from postgres")
		return nil, false
	}

	item.ImageKey = imageKey.String
	item.ProcessedByClip = processedByClip.Bool
	item.ProcessedByQdrant = processedByQdrant.Bool
	item.PublishedOnDaneGov = publishedOnDaneGov.Bool

	return item, true
}

// List returns all items
func (s *PostgresStorage) List() ([]*models.LostItem, error) {
	query := `
	SELECT id, title, description, category, location, found_date,
		   reporting_date, reporting_location,
		   image_url, image_key, status, contact_email, contact_phone,
		   processed_by_clip, processed_by_qdrant, published_on_dane_gov,
		   created_at, updated_at
	FROM lost_items
	ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.LostItem
	for rows.Next() {
		item := &models.LostItem{}
		var processedByClip, processedByQdrant, publishedOnDaneGov sql.NullBool
		var imageKey sql.NullString

		err := rows.Scan(
			&item.ID, &item.Title, &item.Description, &item.Category, &item.Location,
			&item.FoundDate, &item.ReportingDate, &item.ReportingLocation,
			&item.ImageURL, &imageKey, &item.Status, &item.ContactEmail, &item.ContactPhone,
			&processedByClip, &processedByQdrant, &publishedOnDaneGov,
			&item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		item.ImageKey = imageKey.String
		item.ProcessedByClip = processedByClip.Bool
		item.ProcessedByQdrant = processedByQdrant.Bool
		item.PublishedOnDaneGov = publishedOnDaneGov.Bool
		items = append(items, item)
	}

	return items, nil
}

// SaveDataset creates or updates a dataset
func (s *PostgresStorage) SaveDataset(dataset *models.Dataset) error {
	query := `
	INSERT INTO datasets (
		id, title, notes, url, institution_name, email,
		categories, tags, created_at, updated_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
	) ON CONFLICT (id) DO UPDATE SET
		title = EXCLUDED.title,
		notes = EXCLUDED.notes,
		url = EXCLUDED.url,
		institution_name = EXCLUDED.institution_name,
		email = EXCLUDED.email,
		categories = EXCLUDED.categories,
		tags = EXCLUDED.tags,
		updated_at = EXCLUDED.updated_at
	;`

	now := time.Now()
	if dataset.CreatedAt.IsZero() {
		dataset.CreatedAt = now
	}
	dataset.UpdatedAt = now

	_, err := s.db.Exec(query,
		dataset.ID, dataset.Title, dataset.Notes, dataset.URL,
		dataset.InstitutionName, dataset.Email,
		arrayToPostgresArray(dataset.Categories),
		arrayToPostgresArray(dataset.Tags),
		dataset.CreatedAt, dataset.UpdatedAt,
	)

	if err != nil {
		log.Error().Err(err).Msg("Failed to save dataset to postgres")
		return err
	}

	return nil
}

// GetDataset retrieves a dataset by ID
func (s *PostgresStorage) GetDataset(id string) (*models.Dataset, bool) {
	query := `
	SELECT id, title, notes, url, institution_name, email,
		   categories, tags, created_at, updated_at
	FROM datasets WHERE id = $1`

	dataset := &models.Dataset{}

	err := s.db.QueryRow(query, id).Scan(
		&dataset.ID, &dataset.Title, &dataset.Notes, &dataset.URL,
		&dataset.InstitutionName, &dataset.Email,
		postgresArrayToSlice(&dataset.Categories),
		postgresArrayToSlice(&dataset.Tags),
		&dataset.CreatedAt, &dataset.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to get dataset from postgres")
		return nil, false
	}

	return dataset, true
}

// ListDatasets returns all datasets
func (s *PostgresStorage) ListDatasets() ([]*models.Dataset, error) {
	query := `
	SELECT id, title, notes, url, institution_name, email,
		   categories, tags, created_at, updated_at
	FROM datasets
	ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var datasets []*models.Dataset
	for rows.Next() {
		dataset := &models.Dataset{}

		err := rows.Scan(
			&dataset.ID, &dataset.Title, &dataset.Notes, &dataset.URL,
			&dataset.InstitutionName, &dataset.Email,
			postgresArrayToSlice(&dataset.Categories),
			postgresArrayToSlice(&dataset.Tags),
			&dataset.CreatedAt, &dataset.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		datasets = append(datasets, dataset)
	}

	return datasets, nil
}

// AddItemToDataset adds a lost item to a dataset
func (s *PostgresStorage) AddItemToDataset(datasetID, itemID string) error {
	query := `
	INSERT INTO dataset_items (dataset_id, item_id, added_at)
	VALUES ($1, $2, $3)
	ON CONFLICT (dataset_id, item_id) DO NOTHING`

	_, err := s.db.Exec(query, datasetID, itemID, time.Now())
	if err != nil {
		log.Error().Err(err).Str("dataset_id", datasetID).Str("item_id", itemID).Msg("Failed to add item to dataset")
		return err
	}

	return nil
}

// RemoveItemFromDataset removes a lost item from a dataset
func (s *PostgresStorage) RemoveItemFromDataset(datasetID, itemID string) error {
	query := `DELETE FROM dataset_items WHERE dataset_id = $1 AND item_id = $2`

	_, err := s.db.Exec(query, datasetID, itemID)
	if err != nil {
		log.Error().Err(err).Str("dataset_id", datasetID).Str("item_id", itemID).Msg("Failed to remove item from dataset")
		return err
	}

	return nil
}

// GetDatasetWithItems retrieves a dataset with all its associated items
func (s *PostgresStorage) GetDatasetWithItems(datasetID string) (*models.DatasetWithItems, error) {
	dataset, found := s.GetDataset(datasetID)
	if !found {
		return nil, fmt.Errorf("dataset not found")
	}

	query := `
	SELECT li.id, li.title, li.description, li.category, li.location, li.found_date,
		   li.reporting_date, li.reporting_location,
		   li.image_url, li.image_key, li.status, li.contact_email, li.contact_phone,
		   li.processed_by_clip, li.processed_by_qdrant, li.published_on_dane_gov,
		   li.created_at, li.updated_at
	FROM lost_items li
	INNER JOIN dataset_items di ON li.id = di.item_id
	WHERE di.dataset_id = $1
	ORDER BY di.added_at DESC`

	rows, err := s.db.Query(query, datasetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.LostItem
	for rows.Next() {
		item := &models.LostItem{}
		var processedByClip, processedByQdrant, publishedOnDaneGov sql.NullBool
		var imageKey sql.NullString

		err := rows.Scan(
			&item.ID, &item.Title, &item.Description, &item.Category, &item.Location,
			&item.FoundDate, &item.ReportingDate, &item.ReportingLocation,
			&item.ImageURL, &imageKey, &item.Status, &item.ContactEmail, &item.ContactPhone,
			&processedByClip, &processedByQdrant, &publishedOnDaneGov,
			&item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		item.ImageKey = imageKey.String
		item.ProcessedByClip = processedByClip.Bool
		item.ProcessedByQdrant = processedByQdrant.Bool
		item.PublishedOnDaneGov = publishedOnDaneGov.Bool
		items = append(items, item)
	}

	return &models.DatasetWithItems{
		Dataset: *dataset,
		Items:   items,
	}, nil
}

// GetItemsByDataset retrieves all items associated with a specific dataset
func (s *PostgresStorage) GetItemsByDataset(datasetID string) ([]*models.LostItem, error) {
	query := `
	SELECT li.id, li.title, li.description, li.category, li.location, li.found_date,
		   li.reporting_date, li.reporting_location,
		   li.image_url, li.image_key, li.status, li.contact_email, li.contact_phone,
		   li.processed_by_clip, li.processed_by_qdrant, li.published_on_dane_gov,
		   li.created_at, li.updated_at
	FROM lost_items li
	INNER JOIN dataset_items di ON li.id = di.item_id
	WHERE di.dataset_id = $1
	ORDER BY di.added_at DESC`

	rows, err := s.db.Query(query, datasetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.LostItem
	for rows.Next() {
		item := &models.LostItem{}
		var processedByClip, processedByQdrant, publishedOnDaneGov sql.NullBool
		var imageKey sql.NullString

		err := rows.Scan(
			&item.ID, &item.Title, &item.Description, &item.Category, &item.Location,
			&item.FoundDate, &item.ReportingDate, &item.ReportingLocation,
			&item.ImageURL, &imageKey, &item.Status, &item.ContactEmail, &item.ContactPhone,
			&processedByClip, &processedByQdrant, &publishedOnDaneGov,
			&item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		item.ImageKey = imageKey.String
		item.ProcessedByClip = processedByClip.Bool
		item.ProcessedByQdrant = processedByQdrant.Bool
		item.PublishedOnDaneGov = publishedOnDaneGov.Bool
		items = append(items, item)
	}

	return items, nil
}

// Helper functions for PostgreSQL array handling
func arrayToPostgresArray(arr []string) interface{} {
	if len(arr) == 0 {
		return []string{}
	}
	return arr
}

func postgresArrayToSlice(dest *[]string) interface{} {
	return dest
}
