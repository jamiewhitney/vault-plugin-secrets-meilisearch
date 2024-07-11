package meilisearch

import (
	"errors"
	"github.com/meilisearch/meilisearch-go"
)

type meilisearchClient struct {
	*meilisearch.Client
}

func newClient(config *meilisearchConfig) (*meilisearchClient, error) {

	if config == nil {
		return nil, errors.New("client configuration was nil")
	}

	if config.ApiKey == "" {
		return nil, errors.New("api key was not defined")

	}

	if config.URL == "" {
		return nil, errors.New("client URL was not defined")
	}

	c := meilisearch.NewClient(meilisearch.ClientConfig{
		Host:   config.URL,
		APIKey: config.ApiKey,
	})

	return &meilisearchClient{c}, nil
}
