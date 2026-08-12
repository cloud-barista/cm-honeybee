package software

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
)

// GetSnaps lists installed snap packages via `snap list`. Returns an empty slice
// (no error) when snap is not installed or none are present, so hosts without
// snapd are handled gracefully.
//
// When showDefaultPackages is false (default), system snaps (base/os/snapd/
// kernel/gadget types — core, core20..24, bare, snapd, ...) are filtered out —
// the snap analogue of the deb/rpm default-package filtering — leaving
// user-installed apps only.
func GetSnaps(showDefaultPackages bool) ([]software.Snap, error) {
	snaps := make([]software.Snap, 0)

	if _, err := exec.LookPath("snap"); err != nil {
		return snaps, nil // snap not installed on this host
	}

	// Force the C locale so the header row and Notes tokens are stable English
	// (some locales translate the `snap list` header). `snap list` also exits
	// non-zero with "No snaps are installed yet." — treated as empty.
	cmd := exec.Command("snap", "list")
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	out, err := cmd.Output()
	if err != nil {
		return snaps, nil
	}

	// Authoritative name->type map from snapd (nil when unavailable).
	types := snapTypes()

	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue // header row / blank
		}
		f := strings.Fields(line)
		if len(f) < 3 {
			continue
		}
		s := software.Snap{Name: f[0], Version: f[1], Revision: f[2]}
		if len(f) >= 4 {
			s.Tracking = f[3]
		}
		if len(f) >= 5 {
			// Strip the trailing "verified"/"starred" marks (e.g. canonical✓, foo*).
			s.Publisher = strings.TrimRight(f[4], "✓✪*")
		}
		if len(f) >= 6 {
			s.Notes = strings.Join(f[5:], " ")
		}

		if !showDefaultPackages && isSystemSnap(s, types) {
			continue
		}
		snaps = append(snaps, s)
	}

	return snaps, nil
}

// snapTypes queries snapd for each snap's type (app / base / os / snapd / kernel
// / gadget) via its local REST API. Returns nil when the API is unavailable, so
// callers fall back to a heuristic. Using the real type avoids hard-coding base
// snap names (core20, core22, ... and any future ones).
func snapTypes() map[string]string {
	transport := &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", "/run/snapd.socket")
		},
	}
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}

	resp, err := client.Get("http://localhost/v2/snaps")
	if err != nil {
		return nil
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var body struct {
		Result []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil
	}

	m := make(map[string]string, len(body.Result))
	for _, s := range body.Result {
		m[s.Name] = s.Type
	}
	return m
}

// isSystemSnap reports whether a snap is a system snap (base/os/snapd/kernel/
// gadget) rather than a user application. It uses snapd's authoritative type
// when available, otherwise falls back to the `snap list` Notes tokens.
func isSystemSnap(s software.Snap, types map[string]string) bool {
	if t, ok := types[s.Name]; ok {
		return t != "app"
	}
	// Fallback: Notes shows base/core/snapd for system snaps.
	for _, n := range strings.FieldsFunc(s.Notes, func(r rune) bool { return r == ',' || r == ' ' }) {
		switch n {
		case "base", "core", "snapd":
			return true
		}
	}
	return false
}
