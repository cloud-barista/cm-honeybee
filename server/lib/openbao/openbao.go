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
	addr  string
	token string
	http  *http.Client
}

var defaultClient *client

// Init wires the OpenBao client from config. Safe to call once at startup; when
// no address is configured OpenBao stays disabled and callers fall back to the
// local store.
func Init() {
	cfg := config.CMHoneybeeConfig.CMHoneybee.OpenBao
	addr := strings.TrimRight(strings.TrimSpace(cfg.Address), "/")
	if addr == "" {
		logger.Println(logger.INFO, false, "OpenBao: not configured; using local secret storage")
		return
	}

	token := strings.TrimSpace(cfg.Token)
	if token == "" && strings.TrimSpace(cfg.TokenFile) != "" {
		b, err := os.ReadFile(strings.TrimSpace(cfg.TokenFile))
		if err != nil {
			logger.Println(logger.ERROR, false, "OpenBao: cannot read token_file: "+err.Error()+
				"; falling back to local secret storage")
			return
		}
		token = strings.TrimSpace(string(b))
	}
	if token == "" {
		logger.Println(logger.ERROR, false, "OpenBao: address set but no token/token_file; "+
			"falling back to local secret storage")
		return
	}

	defaultClient = &client{addr: addr, token: token, http: &http.Client{Timeout: 15 * time.Second}}
	logger.Println(logger.INFO, false, "OpenBao: enabled ("+addr+")")
}

// Enabled reports whether OpenBao is configured and should be used.
func Enabled() bool { return defaultClient != nil }

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
