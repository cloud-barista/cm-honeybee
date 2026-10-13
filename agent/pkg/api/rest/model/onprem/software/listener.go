package software

// Listener is a host process that owns at least one listening socket. It is
// collected for every such process, package-owned or not, so consumers can tell
// which installed software is actually serving on which port.
type Listener struct {
	PID            int32  `json:"pid"`
	Name           string `json:"name"`
	ExecutablePath string `json:"executable_path"`
	// Ports are the local ports of the process's LISTEN sockets, sorted and unique.
	Ports []int `json:"ports"`
	// Package is the OS package owning ExecutablePath; empty when no package owns it.
	Package string `json:"package,omitempty"`
	// Syft is what syft's binary classifier recognized ExecutablePath as, if anything.
	Syft *SyftMatch `json:"syft,omitempty"`
}

// SyftMatch is a syft binary-classifier match for an executable. It is evidence
// for consumers to weigh, not a verdict: the classifier only knows the products
// it has rules for and may miss renamed or unusual builds.
type SyftMatch struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
