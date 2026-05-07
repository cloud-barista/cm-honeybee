package spider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cloud-barista/cm-honeybee/server/lib/config"
)

const defaultTimeout = 60 * time.Second

func endpoint() string {
	return strings.TrimRight(config.CMHoneybeeConfig.CMHoneybee.Spider.Endpoint, "/")
}

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: defaultTimeout}
}

func do(method, path string, body, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(b)
	}

	u := endpoint() + path
	req, err := http.NewRequestWithContext(context.Background(), method, u, reqBody)
	if err != nil {
		return err
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	if user := config.CMHoneybeeConfig.CMHoneybee.Spider.Username; user != "" {
		req.SetBasicAuth(user, config.CMHoneybeeConfig.CMHoneybee.Spider.Password)
	}

	resp, err := newHTTPClient().Do(req)
	if err != nil {
		return fmt.Errorf("spider request failed (%s %s): %w", method, u, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("spider returned %d for %s %s: %s", resp.StatusCode, method, u, string(respBytes))
	}

	if out == nil || len(respBytes) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBytes, out); err != nil {
		return fmt.Errorf("failed to decode spider response (%s %s): %w (body: %s)", method, u, err, string(respBytes))
	}
	return nil
}

func notFoundError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "spider returned 404")
}

func encodePath(s string) string {
	return url.PathEscape(s)
}

// stringList is a small helper that errors out when a required value is empty.
func mustNonEmpty(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New(name + " is empty")
	}
	return nil
}
