package meilisearch

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
)

func getTestBackend(t *testing.T) (*backend, logical.Storage) {
	t.Helper()

	b := newBackend()
	storage := &logical.InmemStorage{}

	config := logical.TestBackendConfig()
	config.StorageView = storage

	if err := b.Setup(context.Background(), config); err != nil {
		t.Fatalf("failed to setup backend: %v", err)
	}

	return b, storage
}

func TestConfig(t *testing.T) {
	b, storage := getTestBackend(t)
	ctx := context.Background()

	// Config should be empty initially.
	resp, err := b.HandleRequest(ctx, &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "config",
		Storage:   storage,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil {
		t.Fatalf("expected nil response, got: %v", resp)
	}

	// Write config.
	resp, err = b.HandleRequest(ctx, &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "config",
		Storage:   storage,
		Data: map[string]interface{}{
			"host":       "http://localhost:7700",
			"master_key": "test-master-key",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil && resp.IsError() {
		t.Fatalf("unexpected error response: %v", resp)
	}

	// Read config back; master_key should not be returned.
	resp, err = b.HandleRequest(ctx, &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "config",
		Storage:   storage,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Data["host"] != "http://localhost:7700" {
		t.Fatalf("expected host http://localhost:7700, got: %v", resp.Data["host"])
	}
	if _, ok := resp.Data["master_key"]; ok {
		t.Fatal("master_key should not be returned in read response")
	}

	// Delete config.
	resp, err = b.HandleRequest(ctx, &logical.Request{
		Operation: logical.DeleteOperation,
		Path:      "config",
		Storage:   storage,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify config is gone.
	resp, err = b.HandleRequest(ctx, &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "config",
		Storage:   storage,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil {
		t.Fatalf("expected nil response after delete, got: %v", resp)
	}
}

func TestConfigValidation(t *testing.T) {
	b, storage := getTestBackend(t)
	ctx := context.Background()

	// Missing master_key should fail.
	resp, err := b.HandleRequest(ctx, &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "config",
		Storage:   storage,
		Data: map[string]interface{}{
			"host": "http://localhost:7700",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || !resp.IsError() {
		t.Fatal("expected error response for missing master_key")
	}

	// Missing host should fail.
	resp, err = b.HandleRequest(ctx, &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "config",
		Storage:   storage,
		Data: map[string]interface{}{
			"master_key": "test-key",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || !resp.IsError() {
		t.Fatal("expected error response for missing host")
	}
}

func TestRoleCRUD(t *testing.T) {
	b, storage := getTestBackend(t)
	ctx := context.Background()

	// Create a role.
	resp, err := b.HandleRequest(ctx, &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "roles/search-only",
		Storage:   storage,
		Data: map[string]interface{}{
			"actions":     "search",
			"indexes":     "movies,books",
			"description": "Search-only access",
			"ttl":         3600,
			"max_ttl":     7200,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil && resp.IsError() {
		t.Fatalf("unexpected error response: %v", resp)
	}

	// Read the role.
	resp, err = b.HandleRequest(ctx, &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "roles/search-only",
		Storage:   storage,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Data["name"] != "search-only" {
		t.Fatalf("expected name search-only, got: %v", resp.Data["name"])
	}
	actions := resp.Data["actions"].([]string)
	if len(actions) != 1 || actions[0] != "search" {
		t.Fatalf("expected actions [search], got: %v", actions)
	}
	indexes := resp.Data["indexes"].([]string)
	if len(indexes) != 2 {
		t.Fatalf("expected 2 indexes, got: %v", indexes)
	}
	if resp.Data["ttl"] != int64(3600) {
		t.Fatalf("expected ttl 3600, got: %v", resp.Data["ttl"])
	}
	if resp.Data["max_ttl"] != int64(7200) {
		t.Fatalf("expected max_ttl 7200, got: %v", resp.Data["max_ttl"])
	}

	// List roles.
	resp, err = b.HandleRequest(ctx, &logical.Request{
		Operation: logical.ListOperation,
		Path:      "roles/",
		Storage:   storage,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	keys := resp.Data["keys"].([]string)
	if len(keys) != 1 || keys[0] != "search-only" {
		t.Fatalf("expected [search-only], got: %v", keys)
	}

	// Update the role.
	resp, err = b.HandleRequest(ctx, &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "roles/search-only",
		Storage:   storage,
		Data: map[string]interface{}{
			"actions": "search,documents.get",
			"indexes": "*",
			"ttl":     1800,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify update.
	resp, err = b.HandleRequest(ctx, &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "roles/search-only",
		Storage:   storage,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	actions = resp.Data["actions"].([]string)
	if len(actions) != 2 {
		t.Fatalf("expected 2 actions after update, got: %v", actions)
	}
	indexes = resp.Data["indexes"].([]string)
	if len(indexes) != 1 || indexes[0] != "*" {
		t.Fatalf("expected indexes [*], got: %v", indexes)
	}
	if resp.Data["ttl"] != int64(1800) {
		t.Fatalf("expected ttl 1800, got: %v", resp.Data["ttl"])
	}

	// Delete role.
	resp, err = b.HandleRequest(ctx, &logical.Request{
		Operation: logical.DeleteOperation,
		Path:      "roles/search-only",
		Storage:   storage,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify deleted.
	resp, err = b.HandleRequest(ctx, &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "roles/search-only",
		Storage:   storage,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil {
		t.Fatalf("expected nil response after delete, got: %v", resp)
	}
}

func TestRoleValidation(t *testing.T) {
	b, storage := getTestBackend(t)
	ctx := context.Background()

	// Missing actions should fail.
	resp, err := b.HandleRequest(ctx, &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "roles/bad",
		Storage:   storage,
		Data: map[string]interface{}{
			"indexes": "*",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || !resp.IsError() {
		t.Fatal("expected error response for missing actions")
	}

	// Missing indexes should fail.
	resp, err = b.HandleRequest(ctx, &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "roles/bad",
		Storage:   storage,
		Data: map[string]interface{}{
			"actions": "search",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || !resp.IsError() {
		t.Fatal("expected error response for missing indexes")
	}
}

func TestCredsRequiresConfig(t *testing.T) {
	b, storage := getTestBackend(t)
	ctx := context.Background()

	// Create a role first.
	_, err := b.HandleRequest(ctx, &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "roles/test",
		Storage:   storage,
		Data: map[string]interface{}{
			"actions": "search",
			"indexes": "*",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Try to read creds without config — should fail.
	_, err = b.HandleRequest(ctx, &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "creds/test",
		Storage:   storage,
	})
	if err == nil {
		t.Fatal("expected error when config is not set")
	}
}

func TestCredsRequiresRole(t *testing.T) {
	b, storage := getTestBackend(t)
	ctx := context.Background()

	// Try to read creds for non-existent role.
	resp, err := b.HandleRequest(ctx, &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "creds/nonexistent",
		Storage:   storage,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || !resp.IsError() {
		t.Fatal("expected error response for missing role")
	}
}

func TestRoleEntryTTL(t *testing.T) {
	role := &roleEntry{
		Name:    "test",
		Actions: []string{"search"},
		Indexes: []string{"*"},
		TTL:     1 * time.Hour,
		MaxTTL:  2 * time.Hour,
	}

	data := role.toResponseData()
	if data["ttl"] != int64(3600) {
		t.Fatalf("expected ttl 3600, got: %v", data["ttl"])
	}
	if data["max_ttl"] != int64(7200) {
		t.Fatalf("expected max_ttl 7200, got: %v", data["max_ttl"])
	}
}
