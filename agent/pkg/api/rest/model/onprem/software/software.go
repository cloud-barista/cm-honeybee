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

// Snap is an installed snap package (from snapd).
type Snap struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Revision    string `json:"revision"`
	Tracking    string `json:"tracking"`               // channel (e.g. latest/stable)
	Publisher   string `json:"publisher"`
	Notes       string `json:"notes,omitempty"`
	Confinement string `json:"confinement,omitempty"`  // strict/classic/devmode
	Base        string `json:"base,omitempty"`         // base snap (e.g. core22)
	BlobPath    string `json:"blob_path,omitempty"`    // on-disk squashfs blob (for offline migration)
	Type        string `json:"type,omitempty"`         // app/base/os/snapd/...
}

// Flatpak is an installed flatpak application (from `flatpak list`).
type Flatpak struct {
	Name          string `json:"name"`
	ApplicationID string `json:"application_id"`
	Version       string `json:"version"`
	Branch        string `json:"branch"`
	Arch          string `json:"arch"`
	Origin        string `json:"origin"`
	OriginURL     string `json:"origin_url"`          // remote repo URL (from `flatpak remotes`)
	Runtime       string `json:"runtime,omitempty"`   // required runtime ref (e.g. org.gnome.Platform/x86_64/50)
	Installation  string `json:"installation"`         // system | user
}
