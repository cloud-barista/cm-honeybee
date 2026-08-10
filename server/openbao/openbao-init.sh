#!/bin/sh
# Initialize + unseal cm-honeybee's OpenBao and enable the KV v2 secrets engine.
#
# Idempotent: safe to run on every (re)start. On first run it initializes
# OpenBao (1 unseal key / threshold 1 — dev/PoC scale), unseals it, enables the
# KV v2 engine at secret/, and writes the root token to honeybee.token for
# cm-honeybee to consume. Afterwards it stays alive as an auto-unseal watcher so
# OpenBao is re-unsealed automatically after a restart.
#
# ⚠ The unseal key + root token are stored in plaintext under
#   data/openbao/secrets/ (openbao-init.json, honeybee.token). This trades the
#   Shamir key-splitting protection for unattended startup — fine for dev/PoC.
#   For production use Cloud KMS auto-unseal (see openbao-config.hcl) or manual
#   unseal, and protect the secrets directory.
set -eu

BAO_ADDR="${BAO_ADDR:-http://openbao-honeybee:8200}"
export BAO_ADDR
SECRETS_DIR=/openbao/secrets
INIT_FILE="$SECRETS_DIR/openbao-init.json"
TOKEN_FILE="$SECRETS_DIR/honeybee.token"

# 1) Wait until OpenBao answers with a real status table (not a connection error).
echo "[openbao-init] waiting for OpenBao at $BAO_ADDR ..."
until bao status 2>/dev/null | grep -qE 'Initialized|Sealed'; do
    sleep 2
done

# 2) Initialize once (single unseal key for unattended dev/PoC startup).
if ! bao status 2>/dev/null | grep -Eq 'Initialized[[:space:]]+true'; then
    echo "[openbao-init] initializing OpenBao"
    bao operator init -key-shares=1 -key-threshold=1 -format=json >"$INIT_FILE"
    chmod 600 "$INIT_FILE"
fi

UNSEAL_KEY=$(grep -A2 'unseal_keys_b64' "$INIT_FILE" | grep -oE '"[A-Za-z0-9+/=]{20,}"' | head -1 | tr -d '"')
ROOT_TOKEN=$(grep -oE '"root_token"[[:space:]]*:[[:space:]]*"[^"]*"' "$INIT_FILE" | sed 's/.*"root_token"[[:space:]]*:[[:space:]]*"//; s/"$//')

# 3) Unseal if sealed.
if bao status 2>/dev/null | grep -Eq 'Sealed[[:space:]]+true'; then
    echo "[openbao-init] unsealing"
    bao operator unseal "$UNSEAL_KEY" >/dev/null
fi

# 4) Enable KV v2 at secret/ (idempotent) and publish the token for cm-honeybee.
export BAO_TOKEN="$ROOT_TOKEN"
if ! bao secrets list 2>/dev/null | grep -q '^secret/'; then
    echo "[openbao-init] enabling KV v2 at secret/"
    bao secrets enable -version=2 -path=secret kv >/dev/null 2>&1 || true
fi
printf '%s' "$ROOT_TOKEN" >"$TOKEN_FILE"
chmod 600 "$TOKEN_FILE"
echo "[openbao-init] ready: unsealed, KV v2 at secret/, token -> honeybee.token"

# 5) Stay alive as an auto-unseal watcher (re-unseal after a restart).
while true; do
    if bao status 2>/dev/null | grep -Eq 'Sealed[[:space:]]+true'; then
        echo "[openbao-init] re-unsealing after seal"
        bao operator unseal "$UNSEAL_KEY" >/dev/null 2>&1 || true
    fi
    sleep 30
done
