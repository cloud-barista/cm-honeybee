package software

type Software struct {
	DEB    []DEB       `json:"deb"`
	RPM    []RPM       `json:"rpm"`
	Docker []Container `json:"docker"`
	Podman []Container `json:"podman"`
}
