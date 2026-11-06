package software

// Listener is a process that owns at least one listening socket on the host's
// network. It is collected for every such process, package-owned or not, so
// consumers can tell which installed software is actually serving on which port.
type Listener struct {
	PID            int32  `json:"pid"`
	Name           string `json:"name"`
	ExecutablePath string `json:"executable_path"`
	// Ports are the local ports of the process's LISTEN sockets, sorted and unique.
	Ports []int `json:"ports"`
	// ListenSockets are the process's listening sockets one by one: its LISTEN
	// TCP sockets and its bound, unconnected UDP sockets, each with the address
	// it is bound to. A process listening only on UDP has these and no Ports.
	// Sorted by protocol, port and address, without duplicates.
	ListenSockets []ListenSocket `json:"listen_sockets,omitempty"`
	// Package is the OS package owning ExecutablePath; empty when no package owns it.
	Package string `json:"package,omitempty"`
	// PackageVersion is the installed version of Package. The DEB and RPM lists
	// leave out packages other packages depend on, which is where a server's
	// binary usually lives (e.g. mysql-server-core-8.0), so it is recorded here.
	PackageVersion string `json:"package_version,omitempty"`
	// Syft is what syft's binary classifier recognized ExecutablePath as, if anything.
	Syft *SyftMatch `json:"syft,omitempty"`
	// ContainerID is the ID of the Docker or Podman container the process runs
	// in; empty for a host process. Only containers on the host's network are
	// collected: the runtime records no ports for them, so their process is the
	// only record of where they listen. ExecutablePath is then a path inside the
	// container, and Package and Syft are left empty.
	ContainerID string `json:"container_id,omitempty"`
}

// ListenSocket is one listening socket of a process.
type ListenSocket struct {
	Protocol string `json:"protocol"` // tcp, udp
	// BindAddress is the local address the socket is bound to, e.g. "0.0.0.0",
	// "::", "127.0.0.1". A wildcard address accepts traffic on every interface.
	BindAddress string `json:"bind_address"`
	Port        int    `json:"port"`
}

// SyftMatch is a syft binary-classifier match for an executable. It is evidence
// for consumers to weigh, not a verdict: the classifier only knows the products
// it has rules for and may miss renamed or unusual builds.
type SyftMatch struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
