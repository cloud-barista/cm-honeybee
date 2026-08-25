package common

// SourceGroupTypeOnprem represents on-premise sources.
const SourceGroupTypeOnprem = "onprem"

// SourceGroupTypeSSH is the legacy on-premise value, treated as on-prem.
const SourceGroupTypeSSH = "ssh"

// SourceGroupTypeCSP represents cb-spider-backed cloud sources.
const SourceGroupTypeCSP = "csp"

// IsOnpremType reports whether the type is on-premise (onprem, legacy ssh, or empty).
func IsOnpremType(t string) bool {
	return t == "" || t == SourceGroupTypeOnprem || t == SourceGroupTypeSSH
}

// IsCSPType reports whether the type is cb-spider-backed cloud.
func IsCSPType(t string) bool {
	return t == SourceGroupTypeCSP
}

// IsValidSourceGroupType reports whether t is an accepted type.
func IsValidSourceGroupType(t string) bool {
	return IsOnpremType(t) || IsCSPType(t)
}

// ResourceTypeVM represents a CSP virtual machine.
const ResourceTypeVM = "vm"

// ResourceTypeK8s represents a CSP Kubernetes cluster.
const ResourceTypeK8s = "k8s"

// ResourceTypeObjectStorage represents a CSP object-storage bucket.
const ResourceTypeObjectStorage = "object_storage"
