package secretsengine

import (
	"context"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	apiKey = "MASTER_KEY"
	url    = "http://localhost:7700"
)

func getTestBackend(tb testing.TB) (*meilisearchBackend, logical.Storage) {
	tb.Helper()

	config := logical.TestBackendConfig()
	config.StorageView = new(logical.InmemStorage)
	config.Logger = hclog.NewNullLogger()
	config.System = logical.TestSystemView()

	b, err := Factory(context.Background(), config)
	if err != nil {
		tb.Fatal(err)
	}

	return b.(*meilisearchBackend), config.StorageView
}

func TestConfig(t *testing.T) {
	b, reqStorage := getTestBackend(t)

	t.Run("Test Configuration", func(t *testing.T) {
		err := testConfigCreate(t, b, reqStorage, map[string]interface{}{
			"api_key": apiKey,
			"url":     url,
		})

		assert.NoError(t, err)

		//err = testConfigRead(t, b, reqStorage, map[string]interface{}{
		//	"api_key": apiKey,
		//	"url":     url,
		//})
		//
		//assert.NoError(t, err)
		//
		//err = testConfigUpdate(t, b, reqStorage, map[string]interface{}{
		//	"api_key": apiKey,
		//	"url":     "http://hashicups:19090",
		//})
		//
		//assert.NoError(t, err)
		//
		//err = testConfigRead(t, b, reqStorage, map[string]interface{}{
		//	"api_key": apiKey,
		//	"url":     "http://hashicups:19090",
		//})

		assert.NoError(t, err)

		//err = testConfigDelete(t, b, reqStorage)

		assert.NoError(t, err)
	})
}

//func testConfigCreate(t *testing.T, b logical.Backend, s logical.Storage, d map[string]interface{}) error {
//	resp, err := b.HandleRequest(context.Background(), &logical.Request{
//		Operation: logical.CreateOperation,
//		Path:      configStoragePath,
//		Data:      d,
//		Storage:   s,
//	})
//
//	if err != nil {
//		return err
//	}
//
//	if resp != nil && resp.IsError() {
//		return resp.Error()
//	}
//	return nil
//}
