package data

import (
	"time"

	"github.com/cloud-barista/cm-honeybee/agent/common"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/data"
)

// GetDataInfo collects data migration information
func GetDataInfo() (data.DataInfo, error) {
	total := time.Now()
	defer func() {
		common.LogElapsed("data", "total", total, "")
	}()

	var dataInfo data.DataInfo

	// Get MinIO data migration info
	start := time.Now()
	minioData, err := GetMinIODataInfo()
	common.LogElapsed("data", "minio", start, "")
	if err == nil {
		dataInfo.MinIO = &minioData
	}

	return dataInfo, nil
}
