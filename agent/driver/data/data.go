package data

import (
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/data"
)

// GetDataInfo collects data migration information
func GetDataInfo() (data.DataInfo, error) {
	var dataInfo data.DataInfo

	// Get MinIO data migration info
	minioData, err := GetMinIODataInfo()
	if err == nil {
		dataInfo.MinIO = &minioData
	}

	return dataInfo, nil
}
