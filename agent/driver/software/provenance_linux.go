//go:build linux && !android

package software

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// getLaunchProvenance determines how the process was started on this host by
// inspecting its cgroup (the most reliable signal: the cgroup names the systemd
// unit actually managing the process) and its working directory.
func getLaunchProvenance(pid int32) launchProvenance {
	// Command-started processes are reproduced as a Type=simple unit by default.
	p := launchProvenance{LaunchType: "command", ServiceType: "simple"}

	if cwd, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid)); err == nil {
		p.WorkingDirectory = cwd
	}

	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cgroup", pid))
	if err != nil {
		p.LaunchType = "unknown"
		return p
	}

	unit := parseSystemdServiceUnit(string(data))
	if unit == "" {
		// Not under a systemd .service cgroup -> started directly (command/other).
		return p
	}

	p.LaunchType = "systemd"
	p.SystemdUnitName = unit

	if out, err := exec.Command("systemctl", "show", "-p", "FragmentPath", "--value", unit).Output(); err == nil {
		p.SystemdUnitPath = strings.TrimSpace(string(out))
	}

	// is-enabled exits non-zero for disabled/static; treat anything but "enabled"
	// as not enabled.
	if out, err := exec.Command("systemctl", "is-enabled", unit).Output(); err == nil {
		if strings.TrimSpace(string(out)) == "enabled" {
			p.SystemdEnabled = true
		}
	}

	// Authoritative service Type / PIDFile from the running unit.
	if out, err := exec.Command("systemctl", "show", "-p", "Type", "--value", unit).Output(); err == nil {
		if t := strings.TrimSpace(string(out)); t != "" {
			p.ServiceType = t
		}
	}
	if out, err := exec.Command("systemctl", "show", "-p", "PIDFile", "--value", unit).Output(); err == nil {
		p.PIDFile = strings.TrimSpace(string(out))
	}

	p.DeclaredEnv = declaredUnitEnv(unit)

	return p
}

// declaredUnitEnv returns the environment the unit declares: Environment=
// entries plus the contents of every EnvironmentFile=. This is what the software
// was configured with, as opposed to /proc/<pid>/environ, which also carries
// everything systemd and the login session injected at start (INVOCATION_ID,
// JOURNAL_STREAM, MEMORY_PRESSURE_*, the desktop session, ...) -- host state that
// means nothing on a migration target.
func declaredUnitEnv(unit string) []string {
	var env []string
	seen := map[string]bool{}

	add := func(entry string) {
		entry = strings.TrimSpace(entry)
		if entry == "" || strings.HasPrefix(entry, "#") {
			return
		}
		key, _, found := strings.Cut(entry, "=")
		if !found || key == "" || seen[key] {
			return
		}
		seen[key] = true
		env = append(env, entry)
	}

	// systemd prints Environment= as a single space-separated, shell-quoted line.
	if out, err := exec.Command("systemctl", "show", "-p", "Environment", "--value", unit).Output(); err == nil {
		for _, entry := range splitQuoted(strings.TrimSpace(string(out))) {
			add(entry)
		}
	}

	// EnvironmentFiles= lines look like "/path/to/file (ignore_errors=no)".
	if out, err := exec.Command("systemctl", "show", "-p", "EnvironmentFiles", "--value", unit).Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			path := strings.TrimSpace(line)
			if idx := strings.LastIndex(path, " ("); idx >= 0 {
				path = strings.TrimSpace(path[:idx])
			}
			if path == "" {
				continue
			}

			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			for _, entry := range strings.Split(string(data), "\n") {
				add(entry)
			}
		}
	}

	return env
}

// splitQuoted splits a systemd-style space-separated list, keeping values that
// are wrapped in single or double quotes intact.
func splitQuoted(s string) []string {
	var parts []string
	var buf strings.Builder
	var quote rune

	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				buf.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
		case r == ' ':
			if buf.Len() > 0 {
				parts = append(parts, buf.String())
				buf.Reset()
			}
		default:
			buf.WriteRune(r)
		}
	}
	if buf.Len() > 0 {
		parts = append(parts, buf.String())
	}

	return parts
}

// isContainerizedProcess reports whether the process runs inside a container
// (Docker, containerd/CRI, Podman, LXC) rather than directly on the host. Such a
// process must not be collected as a host "legacy binary": its files live inside
// the container image, so it is migrated by the container migration path instead.
// Collecting it as a host binary produces an entry that cannot be reproduced
// (nothing to copy, a synthesized/copied unit whose ExecStart points at
// container-internal paths that never start).
func isContainerizedProcess(pid int32) bool {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cgroup", pid))
	if err != nil {
		return false
	}
	return cgroupIndicatesContainer(string(data))
}

// cgroupIndicatesContainer matches the container-instance markers a runtime writes
// into a member process's cgroup path. It deliberately matches the per-container
// forms (e.g. "/docker/<id>", "docker-<id>.scope", "/kubepods...", "libpod-<id>")
// and not host daemon units like "docker.service" or "containerd.service", so the
// runtime daemons themselves are not misclassified as containerized.
func cgroupIndicatesContainer(cgroup string) bool {
	for _, line := range strings.Split(cgroup, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Take the cgroup path (last colon-separated field).
		path := line
		if idx := strings.LastIndex(line, ":"); idx >= 0 {
			path = line[idx+1:]
		}

		for _, marker := range []string{
			"/docker/", "docker-",
			"/kubepods", "cri-containerd-", "containerd-",
			"/crio-", "crio-",
			"/lxc/",
			"libpod-", "/libpod/",
		} {
			if strings.Contains(path, marker) {
				return true
			}
		}
	}

	return false
}

// parseSystemdServiceUnit extracts the managing <name>.service unit from the
// contents of /proc/<pid>/cgroup. Handles cgroup v2 ("0::/system.slice/foo.service")
// and v1 ("N:controllers:/system.slice/foo.service") layouts. Returns "" when the
// process is not managed by a systemd service (e.g. session scopes, no service).
func parseSystemdServiceUnit(cgroup string) string {
	for _, line := range strings.Split(cgroup, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Take the cgroup path (last colon-separated field).
		path := line
		if idx := strings.LastIndex(line, ":"); idx >= 0 {
			path = line[idx+1:]
		}
		if !strings.Contains(path, ".service") {
			continue
		}

		// The managing service is the last ".service" component in the path.
		segments := strings.Split(path, "/")
		for i := len(segments) - 1; i >= 0; i-- {
			if strings.HasSuffix(segments[i], ".service") {
				return segments[i]
			}
		}
	}

	return ""
}
