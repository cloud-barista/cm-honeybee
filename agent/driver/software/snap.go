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

// snapdAPISnap is the subset of snapd's /v2/snaps result cm-honeybee uses.
type snapdAPISnap struct {
	Name            string `json:"name"`
	Version         string `json:"version"`
	Revision        string `json:"revision"`
	TrackingChannel string `json:"tracking-channel"`
	Channel         string `json:"channel"`
	Type            string `json:"type"` // app/base/os/snapd/kernel/gadget
	Confinement     string `json:"confinement"`
	Base            string `json:"base"`
	DevMode         bool   `json:"devmode"`
	JailMode        bool   `json:"jailmode"`
	Publisher       struct {
		Username    string `json:"username"`
		DisplayName string `json:"display-name"`
	} `json:"publisher"`
}

// GetSnaps lists installed snap packages. Returns an empty slice (no error) when
// snap is not installed or none are present.
//
// It reads from snapd's local REST API, which provides the fields needed to
// reproduce a snap on the target (revision, channel, base, confinement) and to
// migrate it offline (the on-disk blob). When the API is unavailable it falls
// back to parsing `snap list`.
//
// When showDefaultPackages is false (default), system snaps (base/os/snapd/
// kernel/gadget) are filtered out.
func GetSnaps(showDefaultPackages bool) ([]software.Snap, error) {
	snaps := make([]software.Snap, 0)

	if _, err := exec.LookPath("snap"); err != nil {
		return snaps, nil // snap not installed on this host
	}

	apiSnaps := snapdSnaps()
	if apiSnaps == nil {
		return getSnapsFromCLI(showDefaultPackages) // API unavailable — fallback
	}

	for _, s := range apiSnaps {
		if !showDefaultPackages && s.Type != "app" {
			continue // base/os/snapd/kernel/gadget are system snaps
		}

		confinement := s.Confinement
		if s.DevMode {
			confinement = "devmode"
		}
		channel := s.TrackingChannel
		if channel == "" {
			channel = s.Channel
		}
		publisher := s.Publisher.Username
		if publisher == "" {
			publisher = s.Publisher.DisplayName
		}

		snaps = append(snaps, software.Snap{
			Name:        s.Name,
			Version:     s.Version,
			Revision:    s.Revision,
			Tracking:    channel,
			Publisher:   publisher,
			Confinement: confinement,
			Base:        s.Base,
			BlobPath:    snapBlobPath(s.Name, s.Revision),
			Type:        s.Type,
		})
	}

	return snaps, nil
}

// snapBlobPath is the on-disk squashfs blob for a snap revision (used for
// offline/air-gapped migration by copying it to the target).
func snapBlobPath(name, revision string) string {
	if name == "" || revision == "" {
		return ""
	}
	path := "/var/lib/snapd/snaps/" + name + "_" + revision + ".snap"
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	return path
}

// snapdSnaps queries snapd's local REST API for installed snaps. Returns nil when
// unavailable so the caller can fall back to `snap list`.
func snapdSnaps() []snapdAPISnap {
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
		Result []snapdAPISnap `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil
	}
	return body.Result
}

// getSnapsFromCLI parses `snap list` when the snapd API is unavailable. Lacks
// confinement/base/blob detail, so those fields stay empty.
func getSnapsFromCLI(showDefaultPackages bool) ([]software.Snap, error) {
	snaps := make([]software.Snap, 0)

	cmd := exec.Command("snap", "list")
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	out, err := cmd.Output()
	if err != nil {
		return snaps, nil
	}

	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
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
			s.Publisher = strings.TrimRight(f[4], "✓✪*")
		}
		if len(f) >= 6 {
			s.Notes = strings.Join(f[5:], " ")
		}
		s.BlobPath = snapBlobPath(s.Name, s.Revision)
		if !showDefaultPackages && isSystemSnapNote(s.Notes, s.Name) {
			continue
		}
		snaps = append(snaps, s)
	}
	return snaps, nil
}

// isSystemSnapNote is the CLI-fallback heuristic (no snapd type available).
func isSystemSnapNote(notes, name string) bool {
	switch name {
	case "core", "core16", "core18", "core20", "core22", "core24", "bare", "snapd":
		return true
	}
	for _, n := range strings.FieldsFunc(notes, func(r rune) bool { return r == ',' || r == ' ' }) {
		switch n {
		case "base", "core", "snapd":
			return true
		}
	}
	return false
}
