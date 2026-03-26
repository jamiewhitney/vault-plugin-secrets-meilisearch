package meilisearch

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const rolesStoragePrefix = "roles/"

type roleEntry struct {
	Name        string        `json:"name"`
	Actions     []string      `json:"actions"`
	Indexes     []string      `json:"indexes"`
	Description string        `json:"description"`
	TTL         time.Duration `json:"ttl"`
	MaxTTL      time.Duration `json:"max_ttl"`
}

func (r *roleEntry) toResponseData() map[string]interface{} {
	return map[string]interface{}{
		"name":        r.Name,
		"actions":     r.Actions,
		"indexes":     r.Indexes,
		"description": r.Description,
		"ttl":         int64(r.TTL.Seconds()),
		"max_ttl":     int64(r.MaxTTL.Seconds()),
	}
}

func pathRoles(b *backend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "roles/" + framework.GenericNameRegex("name"),
			DisplayAttrs: &framework.DisplayAttributes{
				OperationPrefix: "meilisearch",
			},
			Fields: map[string]*framework.FieldSchema{
				"name": {
					Type:        framework.TypeLowerCaseString,
					Description: "The name of the role.",
					Required:    true,
				},
				"actions": {
					Type:        framework.TypeCommaStringSlice,
					Description: `Comma-separated list of Meilisearch actions (e.g. "search", "documents.*", "*").`,
					Required:    true,
				},
				"indexes": {
					Type:        framework.TypeCommaStringSlice,
					Description: `Comma-separated list of Meilisearch index UIDs this key can access (e.g. "movies", "*").`,
					Required:    true,
				},
				"description": {
					Type:        framework.TypeString,
					Description: "A description to set on generated API keys.",
				},
				"ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "Default TTL for generated keys. If unset, uses the system default.",
				},
				"max_ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "Maximum TTL for generated keys. If unset, uses the system max.",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.pathRoleRead,
				},
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.pathRoleWrite,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.pathRoleWrite,
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.pathRoleDelete,
				},
			},
			ExistenceCheck:  b.pathRoleExistenceCheck,
			HelpSynopsis:    "Manage roles for generating Meilisearch API keys.",
			HelpDescription: "This path lets you create, read, update, and delete roles that define how Meilisearch API keys are generated.",
		},
		{
			Pattern: "roles/?$",
			DisplayAttrs: &framework.DisplayAttributes{
				OperationPrefix: "meilisearch",
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ListOperation: &framework.PathOperation{
					Callback: b.pathRoleList,
				},
			},
			HelpSynopsis:    "List configured roles.",
			HelpDescription: "List all roles configured for generating Meilisearch API keys.",
		},
	}
}

func (b *backend) pathRoleExistenceCheck(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	name := data.Get("name").(string)
	entry, err := req.Storage.Get(ctx, rolesStoragePrefix+name)
	if err != nil {
		return false, err
	}
	return entry != nil, nil
}

func getRole(ctx context.Context, s logical.Storage, name string) (*roleEntry, error) {
	entry, err := s.Get(ctx, rolesStoragePrefix+name)
	if err != nil {
		return nil, fmt.Errorf("error reading role: %w", err)
	}
	if entry == nil {
		return nil, nil
	}

	role := new(roleEntry)
	if err := entry.DecodeJSON(role); err != nil {
		return nil, fmt.Errorf("error decoding role: %w", err)
	}
	return role, nil
}

func (b *backend) pathRoleRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	name := data.Get("name").(string)

	role, err := getRole(ctx, req.Storage, name)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, nil
	}

	return &logical.Response{
		Data: role.toResponseData(),
	}, nil
}

func (b *backend) pathRoleWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	name := data.Get("name").(string)

	role, err := getRole(ctx, req.Storage, name)
	if err != nil {
		return nil, err
	}
	if role == nil {
		role = &roleEntry{Name: name}
	}

	if actions, ok := data.GetOk("actions"); ok {
		role.Actions = actions.([]string)
	}
	if indexes, ok := data.GetOk("indexes"); ok {
		role.Indexes = indexes.([]string)
	}
	if desc, ok := data.GetOk("description"); ok {
		role.Description = desc.(string)
	}
	if ttl, ok := data.GetOk("ttl"); ok {
		role.TTL = time.Duration(ttl.(int)) * time.Second
	}
	if maxTTL, ok := data.GetOk("max_ttl"); ok {
		role.MaxTTL = time.Duration(maxTTL.(int)) * time.Second
	}

	if len(role.Actions) == 0 {
		return logical.ErrorResponse("actions is required"), nil
	}
	if len(role.Indexes) == 0 {
		return logical.ErrorResponse("indexes is required"), nil
	}

	entry, err := logical.StorageEntryJSON(rolesStoragePrefix+name, role)
	if err != nil {
		return nil, err
	}
	if err := req.Storage.Put(ctx, entry); err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *backend) pathRoleDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	name := data.Get("name").(string)

	if err := req.Storage.Delete(ctx, rolesStoragePrefix+name); err != nil {
		return nil, fmt.Errorf("error deleting role: %w", err)
	}

	return nil, nil
}

func (b *backend) pathRoleList(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	entries, err := req.Storage.List(ctx, rolesStoragePrefix)
	if err != nil {
		return nil, fmt.Errorf("error listing roles: %w", err)
	}

	return logical.ListResponse(entries), nil
}
