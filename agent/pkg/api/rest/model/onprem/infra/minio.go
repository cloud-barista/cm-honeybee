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
	ServerInfo     *MinIOServerInfo  `json:"server_info,omitempty"`
	Buckets        []MinioBucket     `json:"buckets,omitempty"`
	Opts           map[string]string `json:"opts,omitempty"`
	Errors         []string          `json:"errors"`
}

// MinIOServerInfo represents MinIO server disk information
type MinIOServerInfo struct {
	Servers      int              `json:"servers"`
	Disks        int              `json:"disks"`
	OnlineDisks  int              `json:"online_disks"`
	OfflineDisks int              `json:"offline_disks"`
	StorageInfo  MinIOStorageInfo `json:"storage_info"`
}

// MinIOStorageInfo represents MinIO storage usage
type MinIOStorageInfo struct {
	TotalGB     float64 `json:"total_gb"`
	UsedGB      float64 `json:"used_gb"`
	FreeGB      float64 `json:"free_gb"`
	UsedPercent float64 `json:"used_percent"`
}

// MinioBucket represents a MinIO bucket with its usage
type MinioBucket struct {
	Name        string  `json:"name"`
	SizeGB      float64 `json:"size_gb"`
	Objects     int64   `json:"objects"`
	Versioned   bool    `json:"versioned"`
	Replication bool    `json:"replication"`
}
