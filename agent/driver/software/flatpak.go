package software

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
)

// flatpakColumns is the stable, tab-separated column set collected per app.
const flatpakColumns = "--columns=name,application,version,branch,arch,origin,runtime,installation"

// GetFlatpaks lists installed flatpak applications. Returns an empty slice (no
// error) when flatpak is not installed or none are present.
//
// Both system (/var/lib/flatpak) and per-user (~/.local/share/flatpak) scopes
// are scanned — scanning only as root would miss every user-scope install.
//
// When showDefaultPackages is false (default), only applications are listed
// (--app), excluding the shared runtimes/platforms.
func GetFlatpaks(showDefaultPackages bool) ([]software.Flatpak, error) {
	flatpaks := make([]software.Flatpak, 0)

	if _, err := exec.LookPath("flatpak"); err != nil {
		return flatpaks, nil // flatpak not installed on this host
	}

	remotes := flatpakRemoteURLs()
	seen := make(map[string]bool) // app-id + scope

	appArg := []string{}
	if !showDefaultPackages {
		appArg = []string{"--app"} // applications only (exclude runtimes)
	}

	// System scope.
	sysArgs := append([]string{"list", "--system"}, appArg...)
	sysArgs = append(sysArgs, flatpakColumns)
	appendFlatpaks(&flatpaks, seen, remotes, runFlatpak(sysArgs, ""))

	// Per-user scope: run as each user that has a flatpak install dir.
	for _, u := range usersWithFlatpak() {
		userArgs := append([]string{"list", "--user"}, appArg...)
		userArgs = append(userArgs, flatpakColumns)
		appendFlatpaks(&flatpaks, seen, remotes, runFlatpak(userArgs, u))
	}

	return flatpaks, nil
}

// runFlatpak runs `flatpak <args...>`, optionally as another user (for user-scope
// installs). Returns the output, or "" on error.
func runFlatpak(args []string, asUser string) string {
	var cmd *exec.Cmd
	if asUser != "" && asUser != "root" {
		cmd = exec.Command("sudo", append([]string{"-n", "-u", asUser, "flatpak"}, args...)...)
	} else {
		cmd = exec.Command("flatpak", args...)
	}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func appendFlatpaks(dst *[]software.Flatpak, seen map[string]bool, remotes map[string]string, out string) {
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
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
		appID := col(1)
		installation := col(7)
		key := appID + "|" + installation
		if appID == "" || seen[key] {
			continue
		}
		seen[key] = true

		origin := col(5)
		*dst = append(*dst, software.Flatpak{
			Name:          col(0),
			ApplicationID: appID,
			Version:       col(2),
			Branch:        col(3),
			Arch:          col(4),
			Origin:        origin,
			OriginURL:     remotes[origin],
			Runtime:       col(6),
			Installation:  installation,
		})
	}
}

// usersWithFlatpak returns usernames (including root) that have a per-user
// flatpak app directory, so their user-scope installs can be enumerated.
func usersWithFlatpak() []string {
	users := make([]string, 0)
	homes := []string{"/root"}
	if entries, err := os.ReadDir("/home"); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				homes = append(homes, "/home/"+e.Name())
			}
		}
	}
	for _, home := range homes {
		if _, err := os.Stat(filepath.Join(home, ".local/share/flatpak/app")); err == nil {
			users = append(users, filepath.Base(home))
		}
	}
	return users
}

// flatpakRemoteURLs returns a map of flatpak remote name -> repo URL (from
// `flatpak remotes`), so the migration target can re-add the same remote instead
// of relying on a hard-coded URL.
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
