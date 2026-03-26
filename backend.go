package meilisearch

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const backendHelp = `
The Meilisearch secrets engine dynamically generates API keys for Meilisearch.

After mounting this secrets engine, configure it using the "config" path,
then create roles using the "roles" path. Credentials can then be generated
using the "creds" path.
`

type backend struct {
	*framework.Backend
	lock   sync.RWMutex
	client *meilisearchClient
}

// Factory returns a configured instance of the Meilisearch backend.
func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b := newBackend()
	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}
	return b, nil
}

func newBackend() *backend {
	b := &backend{}

	b.Backend = &framework.Backend{
		Help:        backendHelp,
		BackendType: logical.TypeLogical,
		Paths: framework.PathAppend(
			pathConfig(b),
			pathRoles(b),
			pathCreds(b),
		),
		Secrets: []*framework.Secret{
			secretMeilisearchKey(b),
		},
		Invalidate: b.invalidate,
	}

	return b
}

func (b *backend) invalidate(_ context.Context, key string) {
	if key == "config" {
		b.lock.Lock()
		defer b.lock.Unlock()
		b.client = nil
	}
}

func (b *backend) getClient(ctx context.Context, s logical.Storage) (*meilisearchClient, error) {
	b.lock.RLock()
	if b.client != nil {
		defer b.lock.RUnlock()
		return b.client, nil
	}
	b.lock.RUnlock()

	b.lock.Lock()
	defer b.lock.Unlock()

	// Double-check after acquiring write lock.
	if b.client != nil {
		return b.client, nil
	}

	config, err := getConfig(ctx, s)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, fmt.Errorf("configuration has not been set; use the config/ endpoint first")
	}

	b.client = newClient(config.Host, config.MasterKey)
	return b.client, nil
}
