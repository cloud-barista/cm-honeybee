// Package openbao is cm-honeybee's optional secrets backend. When configured
// (cm-honeybee.openbao.address), SSH access info and CSP credentials are stored
// in OpenBao's KV v2 engine instead of the local SQLite store. Talks to OpenBao
// over its REST API (no SDK dependency), mirroring the spider client style.
package openbao

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/db"
	"github.com/cloud-barista/cm-honeybee/server/lib/config"
	"github.com/cloud-barista/cm-honeybee/server/lib/rsautil"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/jollaman999/utils/logger"
	"gorm.io/gorm"
)

// mount is the KV v2 mount path, enabled at "secret/" by ensureKV during startup.
const mount = "secret"

// ErrNotFound is returned by Get when the path holds no secret.
var ErrNotFound = errors.New("openbao: secret not found")

type client struct {
	addr  string
	token string // root token, loaded/decrypted from the DB at WaitReady
	http  *http.Client
}

var defaultClient *client

// Init wires the OpenBao client from config. Safe to call once at startup; when
// no address is configured OpenBao stays disabled. cm-honeybee always
// self-manages OpenBao (init/unseal/KV-enable) — see WaitReady — persisting the
// unseal key + root token RSA-encrypted in the DB so it can re-unseal after
// restarts.
func Init() {
	cfg := config.CMHoneybeeConfig.CMHoneybee.OpenBao
	addr := strings.TrimRight(strings.TrimSpace(cfg.Address), "/")
	if addr == "" {
		logger.Println(logger.WARN, false, "OpenBao: not configured (cm-honeybee.openbao.address is empty). "+
			"SSH/CSP credentials cannot be stored until OpenBao is configured.")
		return
	}

	defaultClient = &client{
		addr: addr,
		http: &http.Client{Timeout: 15 * time.Second},
	}
	logger.Println(logger.INFO, false, "OpenBao: enabled ("+addr+"), self-managed init/unseal (keys RSA-encrypted in DB)")
}

// Enabled reports whether OpenBao is configured and should be used.
func Enabled() bool { return defaultClient != nil }

// sealState is the subset of /v1/sys/seal-status cm-honeybee cares about.
type sealState struct {
	Initialized bool `json:"initialized"`
	Sealed      bool `json:"sealed"`
}

// sealStatus queries OpenBao's unauthenticated seal-status endpoint.
func (c *client) sealStatus() (sealState, error) {
	var st sealState
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, c.addr+"/v1/sys/seal-status", nil)
	if err != nil {
		return st, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return st, err
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return st, fmt.Errorf("seal-status returned %d: %s", resp.StatusCode, string(b))
	}
	if err := json.Unmarshal(b, &st); err != nil {
		return st, err
	}
	return st, nil
}

// kvReady confirms the KV v2 engine is mounted at secret/ and the token is
// accepted — i.e. secrets can actually be read/written end to end.
func (c *client) kvReady() error {
	raw, err := c.do(http.MethodGet, "/v1/sys/mounts", nil)
	if err != nil {
		return err
	}
	if !strings.Contains(string(raw), `"secret/"`) {
		return errors.New("KV v2 engine not mounted at secret/ yet")
	}
	return nil
}

// WaitReady blocks until OpenBao is reachable, initialized, unsealed, this
// client holds a usable token, and the KV v2 engine is mounted — everything
// cm-honeybee needs to store/read secrets. cm-honeybee stays not-ready
// (readyz 503) until this returns.
//
// It exists because container start order is NOT guaranteed on a host reboot:
// docker restart policies ignore compose depends_on, and on a fresh or lost
// volume OpenBao comes up UNINITIALIZED. cm-honeybee performs 'operator init' +
// unseal + KV-enable itself and persists the unseal key/root token, so it
// recovers on its own after reboots and volume loss with no init sidecar.
// No-op when OpenBao is not configured.
func WaitReady() {
	c := defaultClient
	if c == nil {
		return
	}
	logger.Println(logger.INFO, false, "OpenBao: waiting until initialized, unsealed, token published, and KV ready ...")

	lastReason := ""
	for i := 0; ; i++ {
		reason := c.ensureReady()
		if reason == "" {
			logger.Println(logger.INFO, false, "OpenBao: ready (initialized, unsealed, KV v2 at secret/)")
			return
		}
		// Log on every state change, plus a heartbeat every ~30s so a stuck
		// state stays visible without spamming.
		if reason != lastReason || i%10 == 0 {
			logger.Println(logger.WARN, false, "OpenBao: not ready yet — "+reason)
			lastReason = reason
		}
		time.Sleep(3 * time.Second)
	}
}

// ensureReady drives OpenBao toward a usable state and loads the root token:
// it performs init/unseal/KV-enable itself. Returns "" when ready, else a short
// reason (for logging + retry).
func (c *client) ensureReady() string {
	st, err := c.sealStatus()
	if err != nil {
		return "unreachable: " + err.Error()
	}

	// 1) Initialize if needed.
	if !st.Initialized {
		uk, rt, err := c.doInit()
		if err != nil {
			return "init failed: " + err.Error()
		}
		if err := c.saveInit(uk, rt); err != nil {
			return "init succeeded but persisting keys failed: " + err.Error()
		}
		c.token = rt
		logger.Println(logger.INFO, false, "OpenBao: initialized (unseal key + root token stored RSA-encrypted in DB)")
		if st, err = c.sealStatus(); err != nil {
			return "post-init status: " + err.Error()
		}
	}

	// 2) Unseal if sealed.
	if st.Sealed {
		uk, rt, err := c.loadInit()
		if err != nil {
			return "sealed but cannot read stored unseal key: " + err.Error()
		}
		if err := c.doUnseal(uk); err != nil {
			return "unseal failed: " + err.Error()
		}
		if c.token == "" {
			c.token = rt
		}
		logger.Println(logger.INFO, false, "OpenBao: unsealed")
		if st, err = c.sealStatus(); err != nil {
			return "post-unseal status: " + err.Error()
		}
		if st.Sealed {
			return "still sealed after unseal attempt"
		}
	}

	// 3) Load the root token (already set above when we just initialized).
	if c.token == "" {
		if _, rt, err := c.loadInit(); err == nil {
			c.token = rt
		}
	}
	if c.token == "" {
		return "no root token available (DB)"
	}

	// 4) Ensure the KV v2 engine is mounted and usable.
	if err := c.ensureKV(); err != nil {
		return "enabling KV v2 failed: " + err.Error()
	}
	if err := c.kvReady(); err != nil {
		return "KV not ready: " + err.Error()
	}
	return ""
}

// doInit runs 'operator init' with a single unseal key (threshold 1) for
// unattended startup, returning the base64 unseal key and the root token.
func (c *client) doInit() (unsealKey, rootToken string, err error) {
	body, _ := json.Marshal(map[string]int{"secret_shares": 1, "secret_threshold": 1})
	raw, err := c.do(http.MethodPost, "/v1/sys/init", body)
	if err != nil {
		return "", "", err
	}
	var r struct {
		KeysB64   []string `json:"keys_base64"`
		RootToken string   `json:"root_token"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return "", "", err
	}
	if len(r.KeysB64) == 0 || r.RootToken == "" {
		return "", "", errors.New("init returned no unseal key or root token")
	}
	return r.KeysB64[0], r.RootToken, nil
}

// doUnseal submits a single unseal key.
func (c *client) doUnseal(key string) error {
	body, _ := json.Marshal(map[string]string{"key": key})
	_, err := c.do(http.MethodPost, "/v1/sys/unseal", body)
	return err
}

// ensureKV mounts the KV v2 engine at secret/ if it is not already mounted.
func (c *client) ensureKV() error {
	if err := c.kvReady(); err == nil {
		return nil
	}
	body, _ := json.Marshal(map[string]any{"type": "kv", "options": map[string]string{"version": "2"}})
	_, err := c.do(http.MethodPost, "/v1/sys/mounts/"+mount, body)
	if err != nil && strings.Contains(err.Error(), "already in use") {
		return nil // mounted concurrently — fine
	}
	return err
}

// rsaEncrypt/rsaDecrypt protect the unseal material at rest using honeybee's key
// (common.PubKey to write, common.PrivKey to read), the same key honeybee uses
// to encrypt other secrets it keeps in the DB.
func rsaEncrypt(plain string) (string, error) {
	if common.PubKey == nil {
		return "", errors.New("honeybee public key not loaded")
	}
	ct, err := rsautil.EncryptWithPublicKey([]byte(plain), common.PubKey)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ct), nil
}

func rsaDecrypt(b64 string) (string, error) {
	if common.PrivKey == nil {
		return "", errors.New("honeybee private key not loaded")
	}
	ct, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}
	pt, err := rsautil.DecryptWithPrivateKey(ct, common.PrivKey)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

// saveInit stores the unseal key + root token RSA-encrypted in the DB (single
// row, upserted).
func (c *client) saveInit(unsealKey, rootToken string) error {
	ukEnc, err := rsaEncrypt(unsealKey)
	if err != nil {
		return err
	}
	rtEnc, err := rsaEncrypt(rootToken)
	if err != nil {
		return err
	}
	return db.DB.Save(&model.OpenBaoInit{ID: 1, UnsealKeyEnc: ukEnc, RootTokenEnc: rtEnc}).Error
}

// loadInit returns the stored unseal key + root token by decrypting the
// RSA-encrypted row from the DB. Returns an error when no row exists yet.
func (c *client) loadInit() (unsealKey, rootToken string, err error) {
	var rec model.OpenBaoInit
	if err := db.DB.First(&rec, 1).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", errors.New("no stored unseal key in DB")
		}
		return "", "", err
	}
	if unsealKey, err = rsaDecrypt(rec.UnsealKeyEnc); err != nil {
		return "", "", fmt.Errorf("decrypt unseal key: %w", err)
	}
	if rootToken, err = rsaDecrypt(rec.RootTokenEnc); err != nil {
		return "", "", fmt.Errorf("decrypt root token: %w", err)
	}
	return unsealKey, rootToken, nil
}

// Put writes (overwriting) the key/values at the KV v2 path.
func Put(path string, data map[string]string) error {
	if defaultClient == nil {
		return errors.New("openbao: not enabled")
	}
	body, err := json.Marshal(map[string]any{"data": data})
	if err != nil {
		return err
	}
	_, err = defaultClient.do(http.MethodPost, "/v1/"+mount+"/data/"+path, body)
	return err
}

// Get reads the key/values at the KV v2 path. Returns ErrNotFound if absent.
func Get(path string) (map[string]string, error) {
	if defaultClient == nil {
		return nil, errors.New("openbao: not enabled")
	}
	raw, err := defaultClient.do(http.MethodGet, "/v1/"+mount+"/data/"+path, nil)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data struct {
			Data map[string]string `json:"data"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("openbao: decode response: %w", err)
	}
	return resp.Data.Data, nil
}

// Delete permanently removes all versions of the secret at the path (KV v2
// metadata delete). Absent paths are treated as success.
func Delete(path string) error {
	if defaultClient == nil {
		return errors.New("openbao: not enabled")
	}
	_, err := defaultClient.do(http.MethodDelete, "/v1/"+mount+"/metadata/"+path, nil)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

func (c *client) do(method, path string, body []byte) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, c.addr+path, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Vault-Token", c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openbao request failed (%s %s): %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openbao returned %d for %s %s: %s", resp.StatusCode, method, path, string(respBytes))
	}
	return respBytes, nil
}
