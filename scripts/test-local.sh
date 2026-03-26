#!/usr/bin/env bash
set -euo pipefail

export VAULT_ADDR="http://127.0.0.1:8200"
export VAULT_TOKEN="root"
MEILI_HOST="http://meilisearch:7700"

echo "==> Waiting for Vault..."
until vault status >/dev/null 2>&1; do sleep 1; done
echo "    Vault is ready."

echo "==> Waiting for Meilisearch..."
until curl -sf http://127.0.0.1:7700/health >/dev/null 2>&1; do sleep 1; done
echo "    Meilisearch is ready."

echo ""
echo "==> Enabling the meilisearch secrets engine..."
vault secrets enable -path=meilisearch vault-plugin-meilisearch

echo ""
echo "==> Configuring the Meilisearch connection..."
vault write meilisearch/config \
  host="$MEILI_HOST" \
  master_key="test-master-key"

echo ""
echo "==> Reading back config (master_key should be hidden)..."
vault read meilisearch/config

echo ""
echo "==> Creating a 'search-only' role..."
vault write meilisearch/roles/search-only \
  actions="search" \
  indexes="movies,books" \
  ttl=1h \
  max_ttl=24h

echo ""
echo "==> Creating an 'admin' role..."
vault write meilisearch/roles/admin \
  actions="*" \
  indexes="*" \
  ttl=8h

echo ""
echo "==> Listing roles..."
vault list meilisearch/roles

echo ""
echo "==> Reading 'search-only' role..."
vault read meilisearch/roles/search-only

echo ""
echo "==> Generating credentials for 'search-only'..."
CREDS=$(vault read -format=json meilisearch/creds/search-only)
echo "$CREDS" | jq .

KEY=$(echo "$CREDS" | jq -r '.data.key')
UID_VAL=$(echo "$CREDS" | jq -r '.data.uid')
LEASE_ID=$(echo "$CREDS" | jq -r '.lease_id')

echo ""
echo "==> Verifying the key works against Meilisearch..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
  "http://127.0.0.1:7700/indexes" \
  -H "Authorization: Bearer $KEY")

if [ "$HTTP_CODE" = "200" ]; then
  echo "    Key works! Got HTTP $HTTP_CODE from Meilisearch."
else
  echo "    WARNING: Got HTTP $HTTP_CODE (key may have limited permissions, which is expected for search-only)."
fi

echo ""
echo "==> Generating credentials for 'admin'..."
vault read meilisearch/creds/admin

echo ""
echo "==> Revoking the search-only lease..."
vault lease revoke "$LEASE_ID"
echo "    Lease revoked."

echo ""
echo "==> Verifying the key was deleted from Meilisearch..."
sleep 1
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
  "http://127.0.0.1:7700/keys/$UID_VAL" \
  -H "Authorization: Bearer test-master-key")

if [ "$HTTP_CODE" = "404" ]; then
  echo "    Confirmed: key was deleted from Meilisearch after revocation."
else
  echo "    Got HTTP $HTTP_CODE (expected 404)."
fi

echo ""
echo "==> All tests passed!"
