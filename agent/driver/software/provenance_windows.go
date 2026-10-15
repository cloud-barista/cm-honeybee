//go:build windows

package software

// getLaunchProvenance is not implemented on Windows; launch provenance detection
// is Linux-specific (cgroup/systemd based).
func getLaunchProvenance(_ int32) launchProvenance {
	return launchProvenance{LaunchType: "unknown", ServiceType: "simple"}
}

// isContainerizedProcess is Linux-specific (cgroup based); on Windows there is no
// equivalent host-side cgroup signal, so nothing is treated as containerized.
func isContainerizedProcess(_ int32) bool {
	return false
}

// containerIDOfProcess is Linux-specific (cgroup based); see isContainerizedProcess.
func containerIDOfProcess(_ int32) string {
	return ""
}

// sharesHostNetwork is Linux-specific (network namespace based); see
// isContainerizedProcess.
func sharesHostNetwork(_ int32) bool {
	return false
}
