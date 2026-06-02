//go:build windows

package software

// getLaunchProvenance is not implemented on Windows; launch provenance detection
// is Linux-specific (cgroup/systemd based).
func getLaunchProvenance(_ int32) launchProvenance {
	return launchProvenance{LaunchType: "unknown", ServiceType: "simple"}
}
