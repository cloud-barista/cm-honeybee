package infra

// MinIO represents MinIO object storage information
type MinIO struct {
	Version         string            `json:"version"`
	ConfigPath      string            `json:"config_path"`
	RootUser        string            `json:"root_user"`
	Volumes         string            `json:"volumes"`
	Address         string            `json:"address"`
	ConsoleAddress  string            `json:"console_address"`
	ProcessInfo     string            `json:"process_info"`
	ServerInfo      *MinIOServerInfo  `json:"server_info"`
	Buckets         []MinioBucket     `json:"buckets"`
	Opts            map[string]string `json:"opts"`
	CORSAllowOrigin []string          `json:"cors_allow_origin"` // Server-level CORS configuration
	Errors          []string          `json:"errors"`
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
	Name        string            `json:"name"`
	SizeGB      float64           `json:"size_gb"`
	Versioned   bool              `json:"versioned"`
	Replication bool              `json:"replication"`
	ObjectLock  bool              `json:"object_lock"`
	Encryption  *BucketEncryption `json:"encryption"`
	Lifecycle   *BucketLifecycle  `json:"lifecycle"`
	Tags        map[string]string `json:"tags"`
}

// BucketEncryption represents bucket encryption configuration
type BucketEncryption struct {
	Enabled bool   `json:"enabled"`
	Type    string `json:"type"` // e.g., "SSE-S3", "SSE-KMS"
}

// BucketLifecycle represents bucket lifecycle configuration
type BucketLifecycle struct {
	Enabled bool `json:"enabled"`
	Rules   int  `json:"rules_count"` // Number of lifecycle rules
}
