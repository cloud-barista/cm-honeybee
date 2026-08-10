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

	defaultClient = &client{
		addr:      addr,
		token:     strings.TrimSpace(cfg.Token),
		tokenFile: strings.TrimSpace(cfg.TokenFile),
		http:      &http.Client{Timeout: 15 * time.Second},
	}
	logger.Println(logger.INFO, false, "OpenBao: enabled ("+addr+")")
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
// docker restart policies ignore compose depends_on, so cm-honeybee can start
// before the openbao-init sidecar has unsealed OpenBao. On a fresh or lost
// volume OpenBao also comes up UNINITIALIZED until the sidecar runs
// 'operator init' + unseal. Rather than trust start ordering (or fail secret
// operations in that window), cm-honeybee verifies the full chain itself.
// No-op when OpenBao is not configured.
func WaitReady() {
	c := defaultClient
	if c == nil {
		return
	}
	logger.Println(logger.INFO, false, "OpenBao: waiting until initialized, unsealed, token published, and KV ready ...")

	lastReason := ""
	for i := 0; ; i++ {
		reason := ""
		st, err := c.sealStatus()
		switch {
		case err != nil:
			reason = "unreachable: " + err.Error()
		case !st.Initialized:
			reason = "not initialized yet (waiting for openbao-init to run 'operator init')"
		case st.Sealed:
			reason = "sealed (waiting for unseal)"
		default:
			if err := c.reloadToken(); err != nil {
				reason = "token not available yet: " + err.Error()
			} else if err := c.kvReady(); err != nil {
				reason = "KV not ready: " + err.Error()
			}
		}

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
