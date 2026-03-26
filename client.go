package meilisearch

import (
	"context"
	"fmt"
	"time"

	ms "github.com/meilisearch/meilisearch-go"
)

// meilisearchClient wraps the official Meilisearch Go SDK for key management.
type meilisearchClient struct {
	client ms.ServiceManager
}

func newClient(host, masterKey string) *meilisearchClient {
	client := ms.New(host, ms.WithAPIKey(masterKey))
	return &meilisearchClient{client: client}
}

// CreateKey creates a new API key in Meilisearch.
func (c *meilisearchClient) CreateKey(ctx context.Context, name, description string, actions, indexes []string, expiresAt *time.Time) (*ms.Key, error) {
	req := &ms.Key{
		Name:        name,
		Description: description,
		Actions:     actions,
		Indexes:     indexes,
	}
	if expiresAt != nil {
		req.ExpiresAt = *expiresAt
	}

	key, err := c.client.CreateKeyWithContext(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create key: %w", err)
	}
	return key, nil
}

// GetKey retrieves a single API key by UID.
func (c *meilisearchClient) GetKey(ctx context.Context, uid string) (*ms.Key, error) {
	key, err := c.client.GetKeyWithContext(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get key: %w", err)
	}
	return key, nil
}

// DeleteKey permanently removes an API key by UID.
func (c *meilisearchClient) DeleteKey(ctx context.Context, uid string) error {
	_, err := c.client.DeleteKeyWithContext(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}
	return nil
}
