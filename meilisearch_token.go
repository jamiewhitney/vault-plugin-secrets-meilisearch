package secretsengine

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

// hashiCupsToken defines a secret for the HashiCups token
type hashiCupsToken struct {
	UserID  int    `json:"user_id"`
	ApiKey  string `json:"api_key"`
	TokenID string `json:"token_id"`
	Token   string `json:"token"`
}

// hashiCupsToken defines a secret to store for a given role
// and how it should be revoked or renewed.
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
	response, err := c.CreateKey(&meilisearch.Key{
		Name:      username.TokenID,
		Actions:   username.Actions,
		Indexes:   username.Indexes,
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
		ExpiresAt: time.Time{},
	})
	if err != nil {
		return nil, fmt.Errorf("error creating HashiCups token: %w", err)
	}

	tokenID := uuid.New().String()

	return &hashiCupsToken{
		ApiKey:  response.UID,
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
	tokenRaw, ok := req.Secret.InternalData["api_key"]
	if ok {
		token, ok = tokenRaw.(string)
		if !ok {
			return nil, fmt.Errorf("invalid value for token in secret internal data")
		}
	}

	if err := b.deleteToken(ctx, client, token); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("no user token workflow implemented")
}

func (b *meilisearchBackend) deleteToken(ctx context.Context, c *meilisearchClient, token string) error {
	response, err := c.DeleteKey(token)
	if err != nil {
		return errors.New("")
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
