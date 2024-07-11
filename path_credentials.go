package meilisearch

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

func pathCredentials(b *meilisearchBackend) *framework.Path {
	return &framework.Path{
		Pattern: "creds/" + framework.GenericNameRegex("name"),
		Fields: map[string]*framework.FieldSchema{
			"name": {
				Type:        framework.TypeLowerCaseString,
				Description: "Name of the role",
				Required:    true,
			},
		},
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.pathCredentialsRead,
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathConfigRead,
			},
		},
	}
}

func (b *meilisearchBackend) createToken(ctx context.Context, s logical.Storage, roleEntry *meilisearchRoleEntry) (*hashiCupsToken, error) {
	client, err := b.getClient(ctx, s)
	if err != nil {
		return nil, err
	}

	var token *hashiCupsToken

	token, err = createToken(ctx, client, roleEntry)
	if err != nil {
		return nil, fmt.Errorf("error creating HashiCups token: %w", err)
	}

	if token == nil {
		return nil, fmt.Errorf("error creating HashiCups token")
	}

	return token, nil
}
func (b *meilisearchBackend) createApiKey(ctx context.Context, req *logical.Request, role *meilisearchRoleEntry) (*logical.Response, error) {
	token, err := b.createToken(ctx, req.Storage, role)
	if err != nil {
		return nil, err
	}
	resp := b.Secret(meilisearchTokenType).Response(map[string]interface{}{
		"token":    token.Token,
		"token_id": token.TokenID,
		"api_key":  token.ApiKey,
	}, map[string]interface{}{
		"token": token.Token,
		"role":  role.Name,
	})

	if role.TTL > 0 {
		resp.Secret.TTL = role.TTL
	}

	if role.MaxTTL > 0 {
		resp.Secret.MaxTTL = role.MaxTTL
	}

	return resp, nil
}

func (b *meilisearchBackend) pathCredentialsRead(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	roleName := d.Get("name").(string)

	roleEntry, err := b.getRole(ctx, req.Storage, roleName)
	if err != nil {
		return nil, fmt.Errorf("error retrieving role: %w", err)
	}

	if roleEntry == nil {
		return nil, errors.New("error retrieving role: role is nil")
	}

	return b.createApiKey(ctx, req, roleEntry)
}
