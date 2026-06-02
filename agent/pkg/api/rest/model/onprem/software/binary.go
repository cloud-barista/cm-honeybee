package software

type ConfigFile struct {
	Path   string `json:"path"`
	Source string `json:"source"` // flag | openfd
}

type Binary struct {
	PID              int32        `json:"pid"`
	Name             string       `json:"name"`
	Version          string       `json:"version"`
	ConnectionStatus string       `json:"connection_status"`
	Cmdline          string       `json:"cmdline"`
	CmdlineSlice     []string     `json:"cmdline_slice"`
	ExecutablePath   string       `json:"executable_path"`
	Environ          []string     `json:"environ"`
	UIDs             []int32      `json:"uids"`
	GIDs             []int32      `json:"gids"`
	Static           bool         `json:"static"`
	Libraries        []string     `json:"libraries"`
	LibraryPaths     []string     `json:"library_paths"`
	OpenFilePaths    []string     `json:"open_file_paths"`
	ConfigFiles      []ConfigFile `json:"config_files"`
	DataDirs         []string     `json:"data_dirs"`
	IsWine           bool         `json:"is_wine"`
	WinePrefix       string       `json:"wine_prefix"`
	// Launch provenance: how the process was started on this host.
	LaunchType       string `json:"launch_type"` // "systemd" | "command" | "unknown"
	SystemdUnitName  string `json:"systemd_unit_name"`
	SystemdUnitPath  string `json:"systemd_unit_path"`
	SystemdEnabled   bool   `json:"systemd_enabled"`
	WorkingDirectory string `json:"working_directory"`
}
