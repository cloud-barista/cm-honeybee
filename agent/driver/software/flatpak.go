package software

import (
	"os/exec"
	"strings"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
)

// GetFlatpaks lists installed flatpak applications via `flatpak list`. Returns an
// empty slice (no error) when flatpak is not installed or none are present.
func GetFlatpaks() ([]software.Flatpak, error) {
	flatpaks := make([]software.Flatpak, 0)

	if _, err := exec.LookPath("flatpak"); err != nil {
		return flatpaks, nil // flatpak not installed on this host
	}

	// Explicit columns give a stable, tab-separated, header-less output.
	out, err := exec.Command("flatpak", "list", "--app",
		"--columns=name,application,version,branch,arch,origin,installation").Output()
	if err != nil {
		return flatpaks, nil
	}

	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		cols := strings.Split(line, "\t")
		col := func(i int) string {
			if i < len(cols) {
				return strings.TrimSpace(cols[i])
			}
			return ""
		}
		flatpaks = append(flatpaks, software.Flatpak{
			Name:          col(0),
			ApplicationID: col(1),
			Version:       col(2),
			Branch:        col(3),
			Arch:          col(4),
			Origin:        col(5),
			Installation:  col(6),
		})
	}

	return flatpaks, nil
}
