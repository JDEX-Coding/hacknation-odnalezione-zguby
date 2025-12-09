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

// CreateCollectionEnhanced creates collection with OPTIMIZED configuration
func (h *QdrantHandler) CreateCollectionEnhanced(ctx context.Context) error {
	// Check if collection exists
	_, err := h.client.Get(ctx, &qdrant.GetCollectionInfoRequest{
		CollectionName: h.collectionName,
	})

	if err == nil {
		log.Printf("✅ Collection '%s' already exists", h.collectionName)
		return nil
	}

	log.Printf("🔧 Creating collection '%s' with ENHANCED configuration...", h.collectionName)

	// ENHANCED: Optimized HNSW configuration for better accuracy
	m := uint64(32)            // Increased from 16 for better connectivity
	efConstruct := uint64(200) // Increased from 100 for better index quality
	fullScanThreshold := uint64(20000)

	// Create collection with 512 dimensions (FIXED from 384!)
	_, err = h.client.Create(ctx, &qdrant.CreateCollection{
		CollectionName: h.collectionName,
		VectorsConfig: &qdrant.VectorsConfig{
			Config: &qdrant.VectorsConfig_Params{
				Params: &qdrant.VectorParams{
					Size:     512, // CRITICAL: Must match CLIP embedding dimension!
					Distance: qdrant.Distance_Cosine,
				},
			},
		},
		HnswConfig: &qdrant.HnswConfigDiff{
			M:                 &m,
			EfConstruct:       &efConstruct,
			FullScanThreshold: &fullScanThreshold,
		},
		// Enable on-disk payload for better memory efficiency
		OnDiskPayload: boolPtr(true),
	})

	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	log.Printf("✅ Collection '%s' created successfully", h.collectionName)
	log.Printf("   📐 Vector size: 512 (CLIP ViT-B/32)")
	log.Printf("   🎯 Distance: Cosine")
	log.Printf("   🔧 HNSW m: %d (high connectivity)", m)
	log.Printf("   🔧 HNSW ef_construct: %d (high quality)", efConstruct)

	return nil
}

// UpsertEmbedding inserts or updates an embedding with associated payload
func (h *QdrantHandler) UpsertEmbedding(ctx context.Context, embedding []float32, payload LostItemPayload) (string, error) {
	// Validate embedding dimension
	if len(embedding) != 512 {
		return "", fmt.Errorf("invalid embedding dimension: %d, expected 512", len(embedding))
	}

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

	log.Printf("✅ Upserted embedding with ID: %s", payload.ItemID)
	return payload.ItemID, nil
}

// SearchSimilarEnhanced performs enhanced vector search with filters
func (h *QdrantHandler) SearchSimilarEnhanced(
	ctx context.Context,
	queryEmbedding []float32,
	limit uint64,
	scoreThreshold float32,
	filter *EnhancedFilter,
) ([]SearchResult, error) {

	// Validate embedding dimension
	if len(queryEmbedding) != 512 {
		return nil, fmt.Errorf("invalid query embedding dimension: %d, expected 512", len(queryEmbedding))
	}

	// Build filter conditions
	var qdrantFilter *qdrant.Filter
	if filter != nil {
		conditions := []*qdrant.Condition{}

		// Category filter
		if filter.Category != "" {
			conditions = append(conditions, &qdrant.Condition{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "category",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: filter.Category,
							},
						},
					},
				},
			})
			log.Printf("🔧 Filter: category=%s", filter.Category)
		}

		// Location filter
		if filter.Location != "" {
			conditions = append(conditions, &qdrant.Condition{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "location",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: filter.Location,
							},
						},
					},
				},
			})
			log.Printf("🔧 Filter: location=%s", filter.Location)
		}

		if len(conditions) > 0 {
			qdrantFilter = &qdrant.Filter{
				Must: conditions,
			}
		}
	}

	// ENHANCED: Higher ef parameter for better search accuracy
	ef := uint64(128) // Search-time accuracy parameter

	// Perform search with enhanced parameters
	searchResult, err := h.pointsClient.Search(ctx, &qdrant.SearchPoints{
		CollectionName: h.collectionName,
		Vector:         queryEmbedding,
		Limit:          limit,
		Filter:         qdrantFilter,
		WithPayload: &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{
				Enable: true,
			},
		},
		ScoreThreshold: &scoreThreshold,
		Params: &qdrant.SearchParams{
			HnswEf: &ef, // ENHANCED: Higher accuracy for search
		},
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

	if len(results) > 0 {
		log.Printf("✅ Found %d similar items (top score: %.4f)", len(results), results[0].Score)
	} else {
		log.Printf("⚠️ No results above threshold %.2f", scoreThreshold)
	}

	return results, nil
}

// SearchSimilar - legacy method for backward compatibility
func (h *QdrantHandler) SearchSimilar(ctx context.Context, queryEmbedding []float32, limit uint64, scoreThreshold float32) ([]SearchResult, error) {
	return h.SearchSimilarEnhanced(ctx, queryEmbedding, limit, scoreThreshold, nil)
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

	log.Printf("✅ Deleted vector with ID: %s", id)
	return nil
}

// GetCollectionInfo returns information about the collection
func (h *QdrantHandler) GetCollectionInfo(ctx context.Context) (*qdrant.CollectionInfo, error) {
	resp, err := h.client.Get(ctx, &qdrant.GetCollectionInfoRequest{
		CollectionName: h.collectionName,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get collection info: %w", err)
	}

	return resp.Result, nil
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
	case int64:
		return &qdrant.Value{
			Kind: &qdrant.Value_IntegerValue{
				IntegerValue: v,
			},
		}
	case float64:
		return &qdrant.Value{
			Kind: &qdrant.Value_DoubleValue{
				DoubleValue: v,
			},
		}
	case bool:
		return &qdrant.Value{
			Kind: &qdrant.Value_BoolValue{
				BoolValue: v,
			},
		}
	default:
		// Try to convert to string
		return &qdrant.Value{
			Kind: &qdrant.Value_StringValue{
				StringValue: fmt.Sprintf("%v", v),
			},
		}
	}
}

func extractPayload(payload map[string]*qdrant.Value) LostItemPayload {
	result := LostItemPayload{}

	if val, ok := payload["item_id"]; ok {
		if strVal, ok := val.GetKind().(*qdrant.Value_StringValue); ok {
			result.ItemID = strVal.StringValue
		}
	}

	if val, ok := payload["title"]; ok {
		if strVal, ok := val.GetKind().(*qdrant.Value_StringValue); ok {
			result.Title = strVal.StringValue
		}
	}

	if val, ok := payload["description"]; ok {
		if strVal, ok := val.GetKind().(*qdrant.Value_StringValue); ok {
			result.Description = strVal.StringValue
		}
	}

	if val, ok := payload["category"]; ok {
		if strVal, ok := val.GetKind().(*qdrant.Value_StringValue); ok {
			result.Category = strVal.StringValue
		}
	}

	if val, ok := payload["location"]; ok {
		if strVal, ok := val.GetKind().(*qdrant.Value_StringValue); ok {
			result.Location = strVal.StringValue
		}
	}

	if val, ok := payload["image_url"]; ok {
		if strVal, ok := val.GetKind().(*qdrant.Value_StringValue); ok {
			result.ImageURL = strVal.StringValue
		}
	}

	if val, ok := payload["contact_info"]; ok {
		if strVal, ok := val.GetKind().(*qdrant.Value_StringValue); ok {
			result.ContactInfo = strVal.StringValue
		}
	}

	return result
}

func getPointID(id *qdrant.PointId) string {
	if uuidID, ok := id.GetPointIdOptions().(*qdrant.PointId_Uuid); ok {
		return uuidID.Uuid
	}
	if numID, ok := id.GetPointIdOptions().(*qdrant.PointId_Num); ok {
		return fmt.Sprintf("%d", numID.Num)
	}
	return ""
}

func boolPtr(b bool) *bool {
	return &b
}
