package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	qdrant "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// QdrantHandler manages interactions with Qdrant vector database
type QdrantHandler struct {
	client         qdrant.CollectionsClient
	pointsClient   qdrant.PointsClient
	conn           *grpc.ClientConn
	collectionName string
}

// LostItemPayload represents the metadata associated with a lost item
type LostItemPayload struct {
	ItemID      string    `json:"item_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Location    string    `json:"location"`
	DateLost    time.Time `json:"date_lost"`
	ImageURL    string    `json:"image_url,omitempty"`
	ContactInfo string    `json:"contact_info,omitempty"`
}

// SearchResult represents a search result from Qdrant
type SearchResult struct {
	ID      string          `json:"id"`
	Score   float32         `json:"score"`
	Payload LostItemPayload `json:"payload"`
}

// NewQdrantHandler creates a new handler for Qdrant operations
func NewQdrantHandler(address, collectionName string) (*QdrantHandler, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Qdrant: %w", err)
	}

	handler := &QdrantHandler{
		client:         qdrant.NewCollectionsClient(conn),
		pointsClient:   qdrant.NewPointsClient(conn),
		conn:           conn,
		collectionName: collectionName,
	}

	return handler, nil
}

// Close closes the connection to Qdrant
func (h *QdrantHandler) Close() error {
	if h.conn != nil {
		return h.conn.Close()
	}
	return nil
}

// CreateCollection creates a new collection in Qdrant if it doesn't exist
func (h *QdrantHandler) CreateCollection(ctx context.Context, vectorSize uint64, distance qdrant.Distance) error {
	// Check if collection exists
	_, err := h.client.Get(ctx, &qdrant.GetCollectionInfoRequest{
		CollectionName: h.collectionName,
	})

	if err == nil {
		log.Printf("Collection '%s' already exists", h.collectionName)
		return nil
	}

	// Create collection
	_, err = h.client.Create(ctx, &qdrant.CreateCollection{
		CollectionName: h.collectionName,
		VectorsConfig: &qdrant.VectorsConfig{
			Config: &qdrant.VectorsConfig_Params{
				Params: &qdrant.VectorParams{
					Size:     vectorSize,
					Distance: distance,
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	log.Printf("Collection '%s' created successfully", h.collectionName)
	return nil
}

// UpsertPoint inserts or updates a point in the collection
func (h *QdrantHandler) UpsertPoint(ctx context.Context, id string, vector []float32, payload LostItemPayload) error {
	pointID := uuid.MustParse(id)

	// Convert payload to map
	payloadMap := map[string]*qdrant.Value{
		"item_id":      {Kind: &qdrant.Value_StringValue{StringValue: payload.ItemID}},
		"title":        {Kind: &qdrant.Value_StringValue{StringValue: payload.Title}},
		"description":  {Kind: &qdrant.Value_StringValue{StringValue: payload.Description}},
		"category":     {Kind: &qdrant.Value_StringValue{StringValue: payload.Category}},
		"location":     {Kind: &qdrant.Value_StringValue{StringValue: payload.Location}},
		"date_lost":    {Kind: &qdrant.Value_StringValue{StringValue: payload.DateLost.Format(time.RFC3339)}},
		"contact_info": {Kind: &qdrant.Value_StringValue{StringValue: payload.ContactInfo}},
	}

	if payload.ImageURL != "" {
		payloadMap["image_url"] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: payload.ImageURL}}
	}

	point := &qdrant.PointStruct{
		Id: &qdrant.PointId{
			PointIdOptions: &qdrant.PointId_Uuid{
				Uuid: pointID.String(),
			},
		},
		Vectors: &qdrant.Vectors{
			VectorsOptions: &qdrant.Vectors_Vector{
				Vector: &qdrant.Vector{
					Data: vector,
				},
			},
		},
		Payload: payloadMap,
	}

	_, err := h.pointsClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: h.collectionName,
		Points:         []*qdrant.PointStruct{point},
		Wait:           boolPtr(true),
	})

	if err != nil {
		return fmt.Errorf("failed to upsert point: %w", err)
	}

	log.Printf("Upserted point %s to collection %s", id, h.collectionName)
	return nil
}

// Search performs a vector similarity search
func (h *QdrantHandler) Search(ctx context.Context, vector []float32, limit uint64) ([]SearchResult, error) {
	response, err := h.pointsClient.Search(ctx, &qdrant.SearchPoints{
		CollectionName: h.collectionName,
		Vector:         vector,
		Limit:          limit,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	results := make([]SearchResult, 0, len(response.Result))
	for _, hit := range response.Result {
		payload := LostItemPayload{}

		if val, ok := hit.Payload["item_id"]; ok {
			if strVal, ok := val.GetKind().(*qdrant.Value_StringValue); ok {
				payload.ItemID = strVal.StringValue
			}
		}
		if val, ok := hit.Payload["title"]; ok {
			if strVal, ok := val.GetKind().(*qdrant.Value_StringValue); ok {
				payload.Title = strVal.StringValue
			}
		}
		if val, ok := hit.Payload["description"]; ok {
			if strVal, ok := val.GetKind().(*qdrant.Value_StringValue); ok {
				payload.Description = strVal.StringValue
			}
		}
		if val, ok := hit.Payload["category"]; ok {
			if strVal, ok := val.GetKind().(*qdrant.Value_StringValue); ok {
				payload.Category = strVal.StringValue
			}
		}
		if val, ok := hit.Payload["location"]; ok {
			if strVal, ok := val.GetKind().(*qdrant.Value_StringValue); ok {
				payload.Location = strVal.StringValue
			}
		}
		if val, ok := hit.Payload["date_lost"]; ok {
			if strVal, ok := val.GetKind().(*qdrant.Value_StringValue); ok {
				if t, err := time.Parse(time.RFC3339, strVal.StringValue); err == nil {
					payload.DateLost = t
				}
			}
		}
		if val, ok := hit.Payload["contact_info"]; ok {
			if strVal, ok := val.GetKind().(*qdrant.Value_StringValue); ok {
				payload.ContactInfo = strVal.StringValue
			}
		}
		if val, ok := hit.Payload["image_url"]; ok {
			if strVal, ok := val.GetKind().(*qdrant.Value_StringValue); ok {
				payload.ImageURL = strVal.StringValue
			}
		}

		var id string
		if uuidID, ok := hit.Id.GetPointIdOptions().(*qdrant.PointId_Uuid); ok {
			id = uuidID.Uuid
		}

		results = append(results, SearchResult{
			ID:      id,
			Score:   hit.Score,
			Payload: payload,
		})
	}

	return results, nil
}

// DeletePoint deletes a point from the collection
func (h *QdrantHandler) DeletePoint(ctx context.Context, id string) error {
	pointID := uuid.MustParse(id)

	_, err := h.pointsClient.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: h.collectionName,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: []*qdrant.PointId{
						{
							PointIdOptions: &qdrant.PointId_Uuid{
								Uuid: pointID.String(),
							},
						},
					},
				},
			},
		},
		Wait: boolPtr(true),
	})

	if err != nil {
		return fmt.Errorf("failed to delete point: %w", err)
	}

	log.Printf("Deleted point %s from collection %s", id, h.collectionName)
	return nil
}

// GetCollectionInfo returns information about the collection
func (h *QdrantHandler) GetCollectionInfo(ctx context.Context) (*qdrant.CollectionInfo, error) {
	response, err := h.client.Get(ctx, &qdrant.GetCollectionInfoRequest{
		CollectionName: h.collectionName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get collection info: %w", err)
	}
	return response.Result, nil
}

// Helper function
func boolPtr(b bool) *bool {
	return &b
}
