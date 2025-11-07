package data

// MinIOData represents MinIO data migration information (only fields required for data migration)
type MinIOData struct {
	RootUser string        `json:"root_user"` // Required for authentication (sensitive)
	Address  string        `json:"address"`   // Required for connection
	Buckets  []MinioBucket `json:"buckets"`   // Required
	Errors   []string      `json:"errors"`
}

// MinioBucket represents a MinIO bucket for data migration
type MinioBucket struct {
	Name       string            `json:"name"`        // Required
	Objects    int64             `json:"objects"`     // Required for planning
	ObjectList []MinIOObjectInfo `json:"object_list"` // Required for migration
	Versioned  bool              `json:"versioned"`   // Required for migration planning
}

// MinIOObjectInfo represents object metadata for integrity verification during migration
type MinIOObjectInfo struct {
	Key          string            `json:"key"`           // Object key (path) - Required
	ETag         string            `json:"etag"`          // ETag for integrity check - Required
	Size         int64             `json:"size"`          // Object size in bytes - Required
	LastModified string            `json:"last_modified"` // Last modified timestamp - Required
	Metadata     map[string]string `json:"metadata"`      // User metadata (includes content-type) - Required
}
