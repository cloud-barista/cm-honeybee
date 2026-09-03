# OpenBao server configuration for cm-honeybee's secrets backend.
#
# Stores connection secrets (SSH password/private key, on-prem k8s kubeconfig)
# and CSP credentials for cm-honeybee (KV v2 engine at secret/). TLS is disabled
# for local/dev; enable it (and a KMS auto-unseal stanza) for production.
# See ../../docs/openbao.md.
#
# Reference: https://openbao.org/docs/configuration/

# Persistent storage — data survives container restarts.
storage "file" {
  path = "/openbao/data"
}

# TCP listener (TLS disabled for local development).
listener "tcp" {
  address     = "0.0.0.0:8200"
  tls_disable = true
}

api_addr = "http://0.0.0.0:8200"

# Disable mlock for container compatibility (IPC_LOCK cap handles memory locking).
disable_mlock = true

# Web UI at http://<host>:8200/ui
ui = true

# ─────────────────────────────────────────────────────────────────────
# (Production) Cloud KMS Auto-Unseal. cm-honeybee unseals this server itself,
# keeping the unseal key RSA-encrypted in its own database; a KMS seal moves
# that key out of the database entirely. Example (AWS KMS):
#
# seal "awskms" {
#   region     = "ap-northeast-2"
#   kms_key_id = "alias/openbao-honeybee-unseal"
# }
# ─────────────────────────────────────────────────────────────────────
