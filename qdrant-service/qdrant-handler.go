package main

import (
	"context"
	"encoding/json"
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

// TODO: Alter to reflect real model
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

// UpsertEmbedding inserts or updates an embedding with associated payload
func (h *QdrantHandler) UpsertEmbedding(ctx context.Context, embedding []float32, payload LostItemPayload) (string, error) {
	// Generate unique ID if not provided
	if payload.ItemID == "" {
		payload.ItemID = uuid.New().String()
	}

	// Convert payload to JSON
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Convert JSON to Qdrant payload format
	var payloadMap map[string]interface{}
	if err := json.Unmarshal(payloadJSON, &payloadMap); err != nil {
		return "", fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	payloadStruct := make(map[string]*qdrant.Value)
	for key, value := range payloadMap {
		payloadStruct[key] = convertToQdrantValue(value)
	}

	// Create point
	point := &qdrant.PointStruct{
		Id: &qdrant.PointId{
			PointIdOptions: &qdrant.PointId_Uuid{
				Uuid: payload.ItemID,
			},
		},
		Vectors: &qdrant.Vectors{
			VectorsOptions: &qdrant.Vectors_Vector{
				Vector: &qdrant.Vector{
					Data: embedding,
				},
			},
		},
		Payload: payloadStruct,
	}

	// Upsert point
	_, err = h.pointsClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: h.collectionName,
		Points:         []*qdrant.PointStruct{point},
		Wait:           boolPtr(true),
	})

	if err != nil {
		return "", fmt.Errorf("failed to upsert point: %w", err)
	}

	log.Printf("Successfully upserted embedding with ID: %s", payload.ItemID)
	return payload.ItemID, nil
}

// SearchSimilar searches for similar vectors in the collection
func (h *QdrantHandler) SearchSimilar(ctx context.Context, queryEmbedding []float32, limit uint64, scoreThreshold float32) ([]SearchResult, error) {
	// Perform search
	searchResult, err := h.pointsClient.Search(ctx, &qdrant.SearchPoints{
		CollectionName: h.collectionName,
		Vector:         queryEmbedding,
		Limit:          limit,
		WithPayload: &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{
				Enable: true,
			},
		},
		ScoreThreshold: &scoreThreshold,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// Convert results
	results := make([]SearchResult, 0, len(searchResult.Result))
	for _, hit := range searchResult.Result {
		payload := extractPayload(hit.Payload)
		results = append(results, SearchResult{
			ID:      getPointID(hit.Id),
			Score:   hit.Score,
			Payload: payload,
		})
	}

	log.Printf("Found %d similar items", len(results))
	return results, nil
}

// GetVectorByID retrieves a vector and its payload by ID
func (h *QdrantHandler) GetVectorByID(ctx context.Context, id string) (*qdrant.RetrievedPoint, error) {
	result, err := h.pointsClient.Get(ctx, &qdrant.GetPoints{
		CollectionName: h.collectionName,
		Ids: []*qdrant.PointId{
			{
				PointIdOptions: &qdrant.PointId_Uuid{
					Uuid: id,
				},
			},
		},
		WithPayload: &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{
				Enable: true,
			},
		},
		WithVectors: &qdrant.WithVectorsSelector{
			SelectorOptions: &qdrant.WithVectorsSelector_Enable{
				Enable: true,
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get vector: %w", err)
	}

	if len(result.Result) == 0 {
		return nil, fmt.Errorf("vector with ID %s not found", id)
	}

	log.Printf("Retrieved vector with ID: %s", id)
	return result.Result[0], nil
}

// DeleteVector deletes a vector by ID
func (h *QdrantHandler) DeleteVector(ctx context.Context, id string) error {
	_, err := h.pointsClient.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: h.collectionName,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: []*qdrant.PointId{
						{
							PointIdOptions: &qdrant.PointId_Uuid{
								Uuid: id,
							},
						},
					},
				},
			},
		},
		Wait: boolPtr(true),
	})

	if err != nil {
		return fmt.Errorf("failed to delete vector: %w", err)
	}

	log.Printf("Deleted vector with ID: %s", id)
	return nil
}

// BatchUpsertEmbeddings upserts multiple embeddings at once
func (h *QdrantHandler) BatchUpsertEmbeddings(ctx context.Context, embeddings [][]float32, payloads []LostItemPayload) ([]string, error) {
	if len(embeddings) != len(payloads) {
		return nil, fmt.Errorf("embeddings and payloads length mismatch")
	}

	points := make([]*qdrant.PointStruct, len(embeddings))
	ids := make([]string, len(embeddings))

	for i, embedding := range embeddings {
		payload := payloads[i]
		if payload.ItemID == "" {
			payload.ItemID = uuid.New().String()
		}
		ids[i] = payload.ItemID

		// Convert payload to JSON
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload at index %d: %w", i, err)
		}

		var payloadMap map[string]interface{}
		if err := json.Unmarshal(payloadJSON, &payloadMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal payload at index %d: %w", i, err)
		}

		payloadStruct := make(map[string]*qdrant.Value)
		for key, value := range payloadMap {
			payloadStruct[key] = convertToQdrantValue(value)
		}

		points[i] = &qdrant.PointStruct{
			Id: &qdrant.PointId{
				PointIdOptions: &qdrant.PointId_Uuid{
					Uuid: payload.ItemID,
				},
			},
			Vectors: &qdrant.Vectors{
				VectorsOptions: &qdrant.Vectors_Vector{
					Vector: &qdrant.Vector{
						Data: embedding,
					},
				},
			},
			Payload: payloadStruct,
		}
	}

	// Batch upsert
	_, err := h.pointsClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: h.collectionName,
		Points:         points,
		Wait:           boolPtr(true),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to batch upsert: %w", err)
	}

	log.Printf("Successfully batch upserted %d embeddings", len(embeddings))
	return ids, nil
}

// Helper functions

func convertToQdrantValue(value interface{}) *qdrant.Value {
	switch v := value.(type) {
	case string:
		return &qdrant.Value{
			Kind: &qdrant.Value_StringValue{
				StringValue: v,
			},
		}
	case float64:
		return &qdrant.Value{
			Kind: &qdrant.Value_DoubleValue{
				DoubleValue: v,
			},
		}
	case int:
		return &qdrant.Value{
			Kind: &qdrant.Value_IntegerValue{
				IntegerValue: int64(v),
			},
		}
	case bool:
		return &qdrant.Value{
			Kind: &qdrant.Value_BoolValue{
				BoolValue: v,
			},
		}
	default:
		// Handle time.Time and other types
		if t, ok := value.(time.Time); ok {
			return &qdrant.Value{
				Kind: &qdrant.Value_StringValue{
					StringValue: t.Format(time.RFC3339),
				},
			}
		}
		return &qdrant.Value{
			Kind: &qdrant.Value_StringValue{
				StringValue: fmt.Sprintf("%v", value),
			},
		}
	}
}

func extractPayload(payload map[string]*qdrant.Value) LostItemPayload {
	result := LostItemPayload{}

	if val, ok := payload["item_id"]; ok {
		if strVal, ok := val.Kind.(*qdrant.Value_StringValue); ok {
			result.ItemID = strVal.StringValue
		}
	}

	if val, ok := payload["title"]; ok {
		if strVal, ok := val.Kind.(*qdrant.Value_StringValue); ok {
			result.Title = strVal.StringValue
		}
	}

	if val, ok := payload["description"]; ok {
		if strVal, ok := val.Kind.(*qdrant.Value_StringValue); ok {
			result.Description = strVal.StringValue
		}
	}

	if val, ok := payload["category"]; ok {
		if strVal, ok := val.Kind.(*qdrant.Value_StringValue); ok {
			result.Category = strVal.StringValue
		}
	}

	if val, ok := payload["location"]; ok {
		if strVal, ok := val.Kind.(*qdrant.Value_StringValue); ok {
			result.Location = strVal.StringValue
		}
	}

	if val, ok := payload["date_lost"]; ok {
		if strVal, ok := val.Kind.(*qdrant.Value_StringValue); ok {
			if t, err := time.Parse(time.RFC3339, strVal.StringValue); err == nil {
				result.DateLost = t
			}
		}
	}

	if val, ok := payload["image_url"]; ok {
		if strVal, ok := val.Kind.(*qdrant.Value_StringValue); ok {
			result.ImageURL = strVal.StringValue
		}
	}

	if val, ok := payload["contact_info"]; ok {
		if strVal, ok := val.Kind.(*qdrant.Value_StringValue); ok {
			result.ContactInfo = strVal.StringValue
		}
	}

	return result
}

func getPointID(id *qdrant.PointId) string {
	if uuid, ok := id.PointIdOptions.(*qdrant.PointId_Uuid); ok {
		return uuid.Uuid
	}
	if num, ok := id.PointIdOptions.(*qdrant.PointId_Num); ok {
		return fmt.Sprintf("%d", num.Num)
	}
	return ""
}

func boolPtr(b bool) *bool {
	return &b
}
