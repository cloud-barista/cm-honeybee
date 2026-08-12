package software

import (
	"os"
	"os/exec"
	"strings"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
)

// GetSnaps lists installed snap packages via `snap list`. Returns an empty slice
// (no error) when snap is not installed or none are present, so hosts without
// snapd are handled gracefully.
//
// When showDefaultPackages is false (default), base/snapd/core "system" snaps
// (core, core20..24, bare, snapd, ...) are filtered out — the snap analogue of
// the deb/rpm default-package filtering — leaving user-installed apps only.
func GetSnaps(showDefaultPackages bool) ([]software.Snap, error) {
	snaps := make([]software.Snap, 0)

	if _, err := exec.LookPath("snap"); err != nil {
		return snaps, nil // snap not installed on this host
	}

	// Force the C locale so the header row and Notes tokens are stable English
	// (some locales translate the `snap list` header), keeping parsing/filtering
	// reliable. `snap list` also exits non-zero with "No snaps are installed
	// yet." — treated as empty.
	cmd := exec.Command("snap", "list")
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	out, err := cmd.Output()
	if err != nil {
		return snaps, nil
	}

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

		if !showDefaultPackages && isSystemSnap(s) {
			continue
		}
		snaps = append(snaps, s)
	}

	return snaps, nil
}

// isSystemSnap reports whether a snap is a base/snapd/core (system) snap rather
// than a user-installed application.
func isSystemSnap(s software.Snap) bool {
	switch s.Name {
	case "core", "core16", "core18", "core20", "core22", "core24", "bare", "snapd":
		return true
	}
	for _, n := range strings.FieldsFunc(s.Notes, func(r rune) bool { return r == ',' || r == ' ' }) {
		switch n {
		case "base", "core", "snapd":
			return true
		}
	}
	return false
}
