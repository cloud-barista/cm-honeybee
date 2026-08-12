package software

type Software struct {
	DEB     []DEB       `json:"deb"`
	RPM     []RPM       `json:"rpm"`
	Snap    []Snap      `json:"snap"`
	Flatpak []Flatpak   `json:"flatpak"`
	Legacy  []Binary    `json:"legacy"`
	Docker  []Container `json:"docker"`
	Podman  []Container `json:"podman"`
}

// Snap is an installed snap package (from `snap list`).
type Snap struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Revision  string `json:"revision"`
	Tracking  string `json:"tracking"`
	Publisher string `json:"publisher"`
	Notes     string `json:"notes"`
}

// Flatpak is an installed flatpak application (from `flatpak list`).
type Flatpak struct {
	Name          string `json:"name"`
	ApplicationID string `json:"application_id"`
	Version       string `json:"version"`
	Branch        string `json:"branch"`
	Arch          string `json:"arch"`
	Origin        string `json:"origin"`
	Installation  string `json:"installation"`
}
