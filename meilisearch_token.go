package meilisearch

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/meilisearch/meilisearch-go"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	meilisearchTokenType = "hashicups_token"
)

type hashiCupsToken struct {
	ApiKey  string `json:"api_key"`
	TokenID string `json:"token_id"`
	Token   string `json:"token"`
}

func (b *meilisearchBackend) meilisearchToken() *framework.Secret {
	return &framework.Secret{
		Type: meilisearchTokenType,
		Fields: map[string]*framework.FieldSchema{
			"api_key": {
				Type:        framework.TypeString,
				Description: "Meilisearch API Key",
			},
		},
		Revoke: b.tokenRevoke,
		Renew:  b.tokenRenew,
	}
}

func createToken(ctx context.Context, c *meilisearchClient, username *meilisearchRoleEntry) (*hashiCupsToken, error) {
	tokenID := uuid.New().String()
	response, err := c.CreateKey(&meilisearch.Key{
		Name:      fmt.Sprintf("vault-%s", tokenID),
		UID:       tokenID,
		Actions:   username.Actions,
		Indexes:   username.Indexes,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		return nil, fmt.Errorf("error creating Meilisearch token: %w", err)
	}

	return &hashiCupsToken{
		ApiKey:  response.Key,
		TokenID: tokenID,
	}, nil
}

// tokenRevoke removes the token from the Vault storage API and calls the client to revoke the token
func (b *meilisearchBackend) tokenRevoke(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	client, err := b.getClient(ctx, req.Storage)
	if err != nil {
		return nil, fmt.Errorf("error getting client: %w", err)
	}

	token := ""
	tokenRaw, ok := req.Secret.InternalData["token_id"]
	if ok {
		token, ok = tokenRaw.(string)
		if !ok {
			return nil, fmt.Errorf("invalid value for token in secret internal data")
		}
	}
	if err := b.deleteToken(ctx, client, token); err != nil {
		return nil, err
	}
	return nil, nil
}

func (b *meilisearchBackend) deleteToken(ctx context.Context, c *meilisearchClient, token string) error {
	response, err := c.DeleteKey(token)
	if err != nil {
		return fmt.Errorf("error deleting Meilisearch token: %w", err)
	}

	if response {
		return errors.New("failed to delete key")
	}
	return nil

}

// tokenRenew calls the client to create a new token and stores it in the Vault storage API
func (b *meilisearchBackend) tokenRenew(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	roleRaw, ok := req.Secret.InternalData["role"]
	if !ok {
		return nil, fmt.Errorf("secret is missing role internal data")
	}

	role := roleRaw.(string)
	roleEntry, err := b.getRole(ctx, req.Storage, role)
	if err != nil {
		return nil, fmt.Errorf("error retrieving role: %w", err)
	}

	if roleEntry == nil {
		return nil, errors.New("error retrieving role: role is nil")
	}

	resp := &logical.Response{Secret: req.Secret}

	if roleEntry.TTL > 0 {
		resp.Secret.TTL = roleEntry.TTL
	}
	if roleEntry.MaxTTL > 0 {
		resp.Secret.MaxTTL = roleEntry.MaxTTL
	}

	return resp, nil
}
