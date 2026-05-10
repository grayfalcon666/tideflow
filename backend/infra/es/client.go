package es

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"

	"tideflow/internal/config"
)

const IndexName = "videos_index"

type Client struct {
	es *elasticsearch.Client
}

func NewClient(conf *config.ElasticsearchConfig) (*Client, error) {
	cfg := elasticsearch.Config{
		Addresses: conf.Addresses,
	}
	if conf.Username != "" && conf.Password != "" {
		cfg.Username = conf.Username
		cfg.Password = conf.Password
	}
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create ES client: %w", err)
	}
	return &Client{es: es}, nil
}

// EnsureIndex creates the videos_index with IK analyzer if it doesn't exist.
func (c *Client) EnsureIndex(ctx context.Context) error {
	// Check if index exists
	res, err := c.es.Indices.Exists([]string{IndexName})
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}
	res.Body.Close()
	if res.StatusCode == 200 {
		return nil // index already exists
	}

	// Create index with mapping
	mapping := `{
		"settings": {
			"number_of_shards": 3,
			"number_of_replicas": 1,
			"analysis": {
				"analyzer": {
					"ik_max": {
						"type": "ik_max_word"
					},
					"ik_smart": {
						"type": "ik_smart"
					}
				}
			}
		},
		"mappings": {
			"properties": {
				"video_id": { "type": "long" },
				"title": {
					"type": "text",
					"analyzer": "ik_max",
					"search_analyzer": "ik_smart"
				},
				"description": {
					"type": "text",
					"analyzer": "ik_max",
					"search_analyzer": "ik_smart"
				},
				"username": {
					"type": "text",
					"analyzer": "ik_max",
					"search_analyzer": "ik_smart"
				},
				"tags": { "type": "keyword" },
				"popularity": { "type": "long" },
				"create_time": { "type": "date", "format": "strict_date_optional_time||epoch_millis" }
			}
		}
	}`

	res, err = c.es.Indices.Create(
		IndexName,
		c.es.Indices.Create.WithBody(strings.NewReader(mapping)),
		c.es.Indices.Create.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("failed to create index: %s", res.String())
	}
	slog.Info("ES index created", "index", IndexName)
	return nil
}

// VideoDoc represents the document stored in ES.
type VideoDoc struct {
	VideoID     uint     `json:"video_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Username    string   `json:"username"`
	Tags        []string `json:"tags"`
	Popularity  int64    `json:"popularity"`
	CreateTime  int64    `json:"create_time"` // unix milliseconds
}

// UpsertVideo inserts or updates a video document in ES.
func (c *Client) UpsertVideo(ctx context.Context, doc *VideoDoc) error {
	body := map[string]interface{}{
		"doc":           doc,
		"doc_as_upsert": true,
	}
	bodyBytes, _ := json.Marshal(body)
	req := esapi.UpdateRequest{
		Index:      IndexName,
		DocumentID: fmt.Sprintf("%d", doc.VideoID),
		Body:       bytes.NewReader(bodyBytes),
	}
	res, err := req.Do(ctx, c.es)
	if err != nil {
		return fmt.Errorf("failed to upsert video %d: %w", doc.VideoID, err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("failed to upsert video %d: %s", doc.VideoID, res.String())
	}
	slog.Info("ES upserted video", "video_id", doc.VideoID)
	return nil
}

// DeleteVideo removes a video document from ES.
func (c *Client) DeleteVideo(ctx context.Context, videoID uint) error {
	req := esapi.DeleteRequest{
		Index:      IndexName,
		DocumentID: fmt.Sprintf("%d", videoID),
	}
	res, err := req.Do(ctx, c.es)
	if err != nil {
		return fmt.Errorf("failed to delete video %d: %w", videoID, err)
	}
	defer res.Body.Close()
	// 404 is acceptable (already deleted)
	if res.IsError() && res.StatusCode != 404 {
		return fmt.Errorf("failed to delete video %d: %s", videoID, res.String())
	}
	slog.Info("ES deleted video", "video_id", videoID)
	return nil
}

// SearchParams defines search options.
type SearchParams struct {
	Query   string
	SortBy  string // "popularity" or "create_time"
	Order   string // "desc" or "asc"
	From    int
	Size    int
}

// SearchResult holds matched video IDs from ES.
type SearchResult struct {
	VideoIDs []uint
	Total    int64
}

// Search performs a multi_match search and returns video IDs.
func (c *Client) Search(ctx context.Context, params SearchParams) (*SearchResult, error) {
	if params.Size == 0 {
		params.Size = 20
	}
	if params.Order == "" {
		params.Order = "desc"
	}

	// Build sort
	sortField := "popularity"
	if params.SortBy == "create_time" {
		sortField = "create_time"
	}

	// Build query body
	queryBody := map[string]interface{}{
		"from": params.From,
		"size": params.Size,
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":    params.Query,
				"fields":   []string{"title^3", "username^2", "description", "tags"},
				"type":     "best_fields",
				"analyzer": "ik_smart",
			},
		},
		"sort": []map[string]interface{}{
			{sortField: map[string]string{"order": params.Order}},
		},
		"_source": []string{"video_id"},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(queryBody); err != nil {
		return nil, err
	}

	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(IndexName),
		c.es.Search.WithBody(&buf),
		c.es.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, fmt.Errorf("ES search failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("ES search error: %s", res.String())
	}

	var response struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source struct {
					VideoID uint `json:"video_id"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode ES response: %w", err)
	}

	result := &SearchResult{
		VideoIDs: make([]uint, 0, len(response.Hits.Hits)),
		Total:    response.Hits.Total.Value,
	}
	for _, hit := range response.Hits.Hits {
		result.VideoIDs = append(result.VideoIDs, hit.Source.VideoID)
	}

	return result, nil
}

// Ping checks ES connectivity.
func (c *Client) Ping(ctx context.Context) error {
	res, err := c.es.Ping(c.es.Ping.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("ES ping failed: %s", res.String())
	}
	return nil
}

// RefreshIndex flushes the index to make documents searchable immediately.
func (c *Client) RefreshIndex(ctx context.Context) error {
	req := esapi.IndicesRefreshRequest{}
	res, err := req.Do(ctx, c.es)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return nil
}

// BulkUpsert performs a bulk index operation for efficiency.
func (c *Client) BulkUpsert(ctx context.Context, docs []*VideoDoc) error {
	if len(docs) == 0 {
		return nil
	}
	var buf bytes.Buffer
	for _, doc := range docs {
		meta := map[string]interface{}{
			"update": map[string]interface{}{
				"_index": IndexName,
				"_id":    fmt.Sprintf("%d", doc.VideoID),
			},
		}
		action, _ := json.Marshal(meta)
		buf.Write(action)
		buf.WriteByte('\n')
		docBytes, _ := json.Marshal(map[string]interface{}{"doc": doc, "doc_as_upsert": true})
		buf.Write(docBytes)
		buf.WriteByte('\n')
	}

	res, err := c.es.Bulk(bytes.NewReader(buf.Bytes()), c.es.Bulk.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("bulk upsert failed: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("bulk upsert error: %s", res.String())
	}
	slog.Info("ES bulk upserted", "count", len(docs))
	return nil
}