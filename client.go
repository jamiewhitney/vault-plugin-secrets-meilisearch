package secretsengine

import (
	"errors"
	"github.com/meilisearch/meilisearch-go"
)

// meilisearchClient creates an object storing
// the client.
type meilisearchClient struct {
	*meilisearch.Client
}

// newClient creates a new client to access HashiCups
// and exposes it for any secrets or roles to use.
func newClient(config *meilisearchConfig) (*meilisearchClient, error) {

	if config == nil {
		return nil, errors.New("client configuration was nil")
	}

	if config.apiKey == "" {
		return nil, errors.New("api key was not defined")

	}

	if config.URL == "" {
		return nil, errors.New("client URL was not defined")
	}

	c := meilisearch.NewClient(meilisearch.ClientConfig{
		Host:   config.URL,
		APIKey: config.apiKey,
	})

	return &meilisearchClient{c}, nil
}
