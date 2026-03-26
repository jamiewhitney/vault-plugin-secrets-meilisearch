package meilisearch

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const configStoragePath = "config"

type configEntry struct {
	Host      string `json:"host"`
	MasterKey string `json:"master_key"`
}

func pathConfig(b *backend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "config",
			DisplayAttrs: &framework.DisplayAttributes{
				OperationPrefix: "meilisearch",
			},
			Fields: map[string]*framework.FieldSchema{
				"host": {
					Type:        framework.TypeString,
					Description: "The URL of the Meilisearch instance (e.g. http://localhost:7700).",
					Required:    true,
				},
				"master_key": {
					Type:        framework.TypeString,
					Description: "The master key used to manage API keys in Meilisearch.",
					Required:    true,
					DisplayAttrs: &framework.DisplayAttributes{
						Sensitive: true,
					},
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.pathConfigRead,
				},
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.pathConfigWrite,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.pathConfigWrite,
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.pathConfigDelete,
				},
			},
			ExistenceCheck:  b.pathConfigExistenceCheck,
			HelpSynopsis:    "Configure the Meilisearch connection.",
			HelpDescription: "Configure the host and master key for communicating with Meilisearch.",
		},
	}
}

func (b *backend) pathConfigExistenceCheck(ctx context.Context, req *logical.Request, _ *framework.FieldData) (bool, error) {
	config, err := getConfig(ctx, req.Storage)
	if err != nil {
		return false, err
	}
	return config != nil, nil
}

func getConfig(ctx context.Context, s logical.Storage) (*configEntry, error) {
	entry, err := s.Get(ctx, configStoragePath)
	if err != nil {
		return nil, fmt.Errorf("error reading configuration: %w", err)
	}
	if entry == nil {
		return nil, nil
	}

	config := new(configEntry)
	if err := entry.DecodeJSON(config); err != nil {
		return nil, fmt.Errorf("error decoding configuration: %w", err)
	}
	return config, nil
}

func (b *backend) pathConfigRead(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	config, err := getConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, nil
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"host": config.Host,
		},
	}, nil
}

func (b *backend) pathConfigWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	config, err := getConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if config == nil {
		config = &configEntry{}
	}

	if host, ok := data.GetOk("host"); ok {
		config.Host = host.(string)
	}
	if masterKey, ok := data.GetOk("master_key"); ok {
		config.MasterKey = masterKey.(string)
	}

	if config.Host == "" {
		return logical.ErrorResponse("host is required"), nil
	}
	if config.MasterKey == "" {
		return logical.ErrorResponse("master_key is required"), nil
	}

	entry, err := logical.StorageEntryJSON(configStoragePath, config)
	if err != nil {
		return nil, err
	}

	if err := req.Storage.Put(ctx, entry); err != nil {
		return nil, err
	}

	// Reset the client so the next request picks up new config.
	b.lock.Lock()
	b.client = nil
	b.lock.Unlock()

	return nil, nil
}

func (b *backend) pathConfigDelete(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	if err := req.Storage.Delete(ctx, configStoragePath); err != nil {
		return nil, fmt.Errorf("error deleting configuration: %w", err)
	}

	b.lock.Lock()
	b.client = nil
	b.lock.Unlock()

	return nil, nil
}
