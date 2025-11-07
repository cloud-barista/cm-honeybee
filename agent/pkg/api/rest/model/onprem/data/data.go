package data

// DataInfo represents all data sources for migration
type DataInfo struct {
	MinIO *MinIOData `json:"minio"`
}
