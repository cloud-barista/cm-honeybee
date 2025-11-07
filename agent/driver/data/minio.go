package data

import (
	"fmt"

	"github.com/cloud-barista/cm-honeybee/agent/driver/infra"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/data"
)

// GetMinIODataInfo collects MinIO data migration information (only required fields)
func GetMinIODataInfo() (data.MinIOData, error) {
	var minioData data.MinIOData

	// Get basic MinIO info from infra driver
	minioInfo, err := infra.GetMinIOInfo()
	if err != nil {
		return minioData, err
	}

	// Extract only fields required for data migration
	minioData.RootUser = minioInfo.RootUser
	minioData.Address = minioInfo.Address
	minioData.Errors = minioInfo.Errors

	// List buckets and collect object information using infra driver
	for _, bucket := range minioInfo.Buckets {
		// Get object list from infra driver
		objects, err := infra.GetMinIOObjectsForBucket(bucket.Name)
		if err != nil {
			minioData.Errors = append(minioData.Errors, fmt.Sprintf("Failed to get objects for bucket %s: %v", bucket.Name, err))
			continue
		}

		// Convert to data model
		var objectList []data.MinIOObjectInfo
		for _, obj := range objects {
			objInfo := data.MinIOObjectInfo{
				Key:          obj["key"].(string),
				ETag:         obj["etag"].(string),
				Size:          obj["size"].(int64),
				LastModified: obj["last_modified"].(string),
				ContentType:  obj["content_type"].(string),
				Metadata:     obj["metadata"].(map[string]string),
			}
			objectList = append(objectList, objInfo)
		}

		dataBucket := data.MinioBucket{
			Name:       bucket.Name,
			Objects:    int64(len(objectList)), // Calculate object count from object list
			ObjectList: objectList,
			Versioned:  bucket.Versioned,
		}
		minioData.Buckets = append(minioData.Buckets, dataBucket)
	}

	return minioData, nil
}
