package infra

// MinIO represents MinIO object storage information
type MinIO struct {
	Version        string            `json:"version"`
	ConfigPath     string            `json:"config_path"`
	RootUser       string            `json:"root_user,omitempty"`
	Volumes        string            `json:"volumes,omitempty"`
	Address        string            `json:"address,omitempty"`
	ConsoleAddress string            `json:"console_address,omitempty"`
	ProcessInfo    string            `json:"process_info,omitempty"`
	StoragePaths   []MinIOStorage    `json:"storage_paths,omitempty"`
	Opts           map[string]string `json:"opts,omitempty"`
	Errors         []string          `json:"errors,omitempty"`
}

// MinIOStorage represents storage volume information
type MinIOStorage struct {
	Path        string  `json:"path"`
	TotalGB     float64 `json:"total_gb"`
	UsedGB      float64 `json:"used_gb"`
	FreeGB      float64 `json:"free_gb"`
	UsedPercent float64 `json:"used_percent"`
}
