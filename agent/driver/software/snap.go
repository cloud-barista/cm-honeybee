package software

import (
	"os/exec"
	"strings"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
)

// GetSnaps lists installed snap packages via `snap list`. Returns an empty slice
// (no error) when snap is not installed or no snaps are present, so hosts without
// snapd are handled gracefully.
func GetSnaps() ([]software.Snap, error) {
	snaps := make([]software.Snap, 0)

	if _, err := exec.LookPath("snap"); err != nil {
		return snaps, nil // snap not installed on this host
	}

	// `snap list` prints a table: Name Version Rev Tracking Publisher Notes.
	// It exits non-zero with "No snaps are installed yet." — treated as empty.
	out, err := exec.Command("snap", "list").Output()
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
		snaps = append(snaps, s)
	}

	return snaps, nil
}
