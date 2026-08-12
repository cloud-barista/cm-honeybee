package software

import (
	"os/exec"
	"strings"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
)

// GetFlatpaks lists installed flatpak applications via `flatpak list`. Returns an
// empty slice (no error) when flatpak is not installed or none are present.
//
// When showDefaultPackages is false (default), only applications are listed
// (--app) so the shared runtimes/platforms (org.freedesktop.Platform,
// org.gnome.Platform, ...) — the flatpak analogue of system/base packages — are
// excluded. When true, runtimes are included as well.
func GetFlatpaks(showDefaultPackages bool) ([]software.Flatpak, error) {
	flatpaks := make([]software.Flatpak, 0)

	if _, err := exec.LookPath("flatpak"); err != nil {
		return flatpaks, nil // flatpak not installed on this host
	}

	// Explicit columns give a stable, tab-separated, header-less output.
	args := []string{"list", "--columns=name,application,version,branch,arch,origin,installation"}
	if !showDefaultPackages {
		args = append(args, "--app") // applications only (exclude runtimes)
	}
	out, err := exec.Command("flatpak", args...).Output()
	if err != nil {
		return flatpaks, nil
	}

	remotes := flatpakRemoteURLs() // remote name -> repo URL (from this host)

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
		origin := col(5)
		flatpaks = append(flatpaks, software.Flatpak{
			Name:          col(0),
			ApplicationID: col(1),
			Version:       col(2),
			Branch:        col(3),
			Arch:          col(4),
			Origin:        origin,
			OriginURL:     remotes[origin],
			Installation:  col(6),
		})
	}

	return flatpaks, nil
}

// flatpakRemoteURLs returns a map of flatpak remote name -> repo URL, read from
// the source host (`flatpak remotes`), so the migration target can re-add the
// same remote instead of relying on a hard-coded URL.
func flatpakRemoteURLs() map[string]string {
	m := make(map[string]string)
	out, err := exec.Command("flatpak", "remotes", "--columns=name,url").Output()
	if err != nil {
		return m
	}
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) >= 2 {
			m[strings.TrimSpace(f[0])] = strings.TrimSpace(f[1])
		}
	}
	return m
}
