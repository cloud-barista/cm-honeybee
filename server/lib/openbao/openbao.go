// Package openbao is cm-honeybee's optional secrets backend. When configured
// (cm-honeybee.openbao.address), SSH access info and CSP credentials are stored
// in OpenBao's KV v2 engine instead of the local SQLite store. Talks to OpenBao
// over its REST API (no SDK dependency), mirroring the spider client style.
package openbao

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/lib/config"
	"github.com/jollaman999/utils/logger"
)

// mount is the KV v2 mount path (enabled at "secret/" by openbao-init).
const mount = "secret"

// ErrNotFound is returned by Get when the path holds no secret.
var ErrNotFound = errors.New("openbao: secret not found")

type client struct {
	addr      string
	token     string
	tokenFile string
	manage    bool   // perform init/unseal/KV-enable ourselves
	initFile  string // where the self-managed unseal key + root token are stored
	http      *http.Client
}

var defaultClient *client

// Init wires the OpenBao client from config. Safe to call once at startup; when
// no address is configured OpenBao stays disabled. When an address IS set, the
// client is created immediately but the token is loaded lazily (it may not
// exist yet — see WaitReady): the openbao-init sidecar can publish it after
// cm-honeybee starts (fresh volume / host reboot).
func Init() {
	cfg := config.CMHoneybeeConfig.CMHoneybee.OpenBao
	addr := strings.TrimRight(strings.TrimSpace(cfg.Address), "/")
	if addr == "" {
		logger.Println(logger.WARN, false, "OpenBao: not configured (cm-honeybee.openbao.address is empty). "+
			"SSH/CSP credentials cannot be stored until OpenBao is configured.")
		return
	}

	initFile := strings.TrimSpace(cfg.InitFile)
	if cfg.Manage && initFile == "" {
		initFile = common.RootPath + "/openbao-init.json"
	}

	defaultClient = &client{
		addr:      addr,
		token:     strings.TrimSpace(cfg.Token),
		tokenFile: strings.TrimSpace(cfg.TokenFile),
		manage:    cfg.Manage,
		initFile:  initFile,
		http:      &http.Client{Timeout: 15 * time.Second},
	}
	mode := "external init/unseal"
	if cfg.Manage {
		mode = "self-managed init/unseal (init_file: " + initFile + ")"
	}
	logger.Println(logger.INFO, false, "OpenBao: enabled ("+addr+"), "+mode)
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

// reloadToken (re)reads the token from token_file when configured, so a token
// written or rotated by the openbao-init sidecar AFTER cm-honeybee started
// (fresh volume, host reboot) is picked up. A static configured token is kept.
func (c *client) reloadToken() error {
	if c.tokenFile == "" {
		if c.token == "" {
			return errors.New("no token or token_file configured")
		}
		return nil
	}
	b, err := os.ReadFile(c.tokenFile)
	if err != nil {
		return err
	}
	t := strings.TrimSpace(string(b))
	if t == "" {
		return errors.New("token_file is empty")
	}
	c.token = t
	return nil
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
// volume OpenBao comes up UNINITIALIZED. In self-managed mode (openbao.manage)
// cm-honeybee performs 'operator init' + unseal + KV-enable itself and persists
// the unseal key/root token, so it recovers on its own after reboots and
// volume loss with no init sidecar. Otherwise it waits for an external
// initializer. No-op when OpenBao is not configured.
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

// ensureReady drives OpenBao toward a usable state and loads a token. In
// self-managed mode it performs init/unseal/KV-enable itself; otherwise it
// waits for an external initializer. Returns "" when ready, else a short reason
// (for logging + retry).
func (c *client) ensureReady() string {
	st, err := c.sealStatus()
	if err != nil {
		return "unreachable: " + err.Error()
	}

	// 1) Initialize (self-managed only).
	if !st.Initialized {
		if !c.manage {
			return "not initialized yet (waiting for external initializer)"
		}
		uk, rt, err := c.doInit()
		if err != nil {
			return "init failed: " + err.Error()
		}
		if err := c.saveInit(uk, rt); err != nil {
			return "init succeeded but persisting keys failed: " + err.Error()
		}
		c.token = rt
		logger.Println(logger.INFO, false, "OpenBao: initialized (unseal key + root token stored at "+c.initFile+")")
		if st, err = c.sealStatus(); err != nil {
			return "post-init status: " + err.Error()
		}
	}

	// 2) Unseal.
	if st.Sealed {
		if !c.manage {
			return "sealed (waiting for external unseal)"
		}
		uk, rt, err := c.loadInit()
		if err != nil {
			return "sealed but cannot read stored unseal key (" + c.initFile + "): " + err.Error()
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

	// 3) Load the token.
	if c.manage {
		if c.token == "" {
			if _, rt, err := c.loadInit(); err == nil {
				c.token = rt
			}
		}
		if c.token == "" {
			return "no root token available (init_file: " + c.initFile + ")"
		}
	} else if err := c.reloadToken(); err != nil {
		return "token not available yet: " + err.Error()
	}

	// 4) Ensure the KV v2 engine is mounted and usable.
	if c.manage {
		if err := c.ensureKV(); err != nil {
			return "enabling KV v2 failed: " + err.Error()
		}
	}
	if err := c.kvReady(); err != nil {
		return "KV not ready: " + err.Error()
	}
	return ""
}

// initFileData is the persisted self-managed unseal material.
type initFileData struct {
	UnsealKey string `json:"unseal_key"`
	RootToken string `json:"root_token"`
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

func (c *client) saveInit(unsealKey, rootToken string) error {
	b, err := json.Marshal(initFileData{UnsealKey: unsealKey, RootToken: rootToken})
	if err != nil {
		return err
	}
	return os.WriteFile(c.initFile, b, 0o600)
}

func (c *client) loadInit() (unsealKey, rootToken string, err error) {
	b, err := os.ReadFile(c.initFile)
	if err != nil {
		return "", "", err
	}
	var d initFileData
	if err := json.Unmarshal(b, &d); err != nil {
		return "", "", err
	}
	if d.UnsealKey == "" {
		return "", "", errors.New("stored init file has no unseal key")
	}
	return d.UnsealKey, d.RootToken, nil
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
