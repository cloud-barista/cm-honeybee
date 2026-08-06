# cm-honeybee + OpenBao (secrets backend)

Brings up **cm-honeybee** together with its own **OpenBao** secrets manager, so
that SSH access info and CSP credentials can be stored in OpenBao (KV v2) instead
of the SQLite DB. This mirrors the OpenBao integration already used by
mc-terrarium / cb-tumblebug.

## Services

| Service        | Role |
|----------------|------|
| `openbao`      | OpenBao server (persistent file storage, KV v2, port 8200) |
| `openbao-init` | One-shot init + unseal + `KV v2 enable` at `secret/`, then an auto-unseal watcher. Publishes the root token to `data/openbao/secrets/honeybee.token`. |
| `cm-honeybee`  | honeybee server (8081 / 8082), wired to reach OpenBao. |

## Usage

```bash
cd server/deployments/docker-compose
docker compose up -d

# OpenBao is initialized, unsealed, and KV v2 is enabled automatically.
docker compose logs -f openbao-init      # watch the bootstrap

# Web UI: http://localhost:8200/ui  (token: data/openbao/secrets/honeybee.token)
```

### Verify

```bash
TOKEN=$(sudo cat data/openbao/secrets/honeybee.token)
docker exec -e BAO_ADDR=http://127.0.0.1:8200 -e BAO_TOKEN="$TOKEN" honeybee-openbao \
  bao kv put secret/csp/aws AWS_ACCESS_KEY_ID=... AWS_SECRET_ACCESS_KEY=...
docker exec -e BAO_ADDR=http://127.0.0.1:8200 -e BAO_TOKEN="$TOKEN" honeybee-openbao \
  bao kv get secret/csp/aws
```

## Secret path convention

Aligned with mc-terrarium's OpenBao layout:

| Kind | Path | Notes |
|------|------|-------|
| CSP credential | `secret/csp/{provider}` | e.g. `secret/csp/aws`, `secret/csp/azure` |
| SSH access     | `secret/ssh/{connectionId}` | per source-group connection |

CSP credential key names follow cb-tumblebug's convention (see
[template.credentials.yaml](https://github.com/cloud-barista/cb-tumblebug/blob/main/init/template.credentials.yaml))
and mc-terrarium's [credential paths](https://github.com/cloud-barista/mc-terrarium/blob/main/deployments/docker-compose/openbao/README.md).

## cm-honeybee wiring

`cm-honeybee` receives:

- `HONEYBEE_VAULT_ADDR=http://openbao:8200`
- `HONEYBEE_VAULT_TOKEN_FILE=/run/openbao/honeybee.token` (mounted read-only)

> The honeybee server code that reads/writes secrets from OpenBao is the next
> step; until it lands these variables are harmless and the container runs as
> before (SQLite-backed).

## Security notes

The `openbao-init` sidecar stores the unseal key + root token in plaintext under
`data/openbao/secrets/` and auto-unseals on restart. This is for **dev / PoC**
convenience and trades away OpenBao's Shamir key-splitting protection.

For production, use one of:

1. **Cloud KMS auto-unseal** — add a `seal "awskms" { … }` stanza to
   `openbao/openbao-config.hcl` and remove the plaintext key. (recommended)
2. **Manual unseal** — remove the `openbao-init` watcher loop and unseal by hand
   after each restart.

Also enable TLS (`tls_disable = false` + certs) and restrict access to the
secrets directory. Runtime data (`data/`) is gitignored.
