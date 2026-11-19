package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	storagemodel "github.com/cloud-barista/cm-beetle/imdl/storage-model"
	serverCommon "github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/jollaman999/utils/logger"
	"github.com/labstack/echo/v4"
)

// errBucketNotCollected means the connection has no CSP bucket stored: it is
// not an object-storage connection of a CSP source group, or it was not
// imported yet.
var errBucketNotCollected = errors.New("no CSP bucket information was collected for this connection; run import/infra first")

// errBucketCollectFailed means the last import of the bucket did not succeed.
// What is stored is then either a result that is no longer current or one
// whose fields could not be read, which would pass for unset values (for
// example encryption reading as off).
var errBucketCollectFailed = errors.New("the last collection of the bucket did not succeed; check the errors of GET .../data and run import/infra again")

// doGetRefinedBucket reads the bucket stored for a connection with the region
// it was found in, empty where the CSP could not tell.
func doGetRefinedBucket(connID string) (*cspBucketRecord, error) {
	saved, err := dao.SavedDataInfoGet(connID)
	if err != nil || saved == nil {
		return nil, errBucketNotCollected
	}
	// Only a CSP bucket's own record is judged by its status: an agent's data
	// record holds no bucket here, whatever became of its import.
	if saved.Status != model.ConnectionInfoStatusSuccess && isCSPBucketRecord(connID, saved) {
		return nil, fmt.Errorf("%w (ConnectionID = %s, Status = %s)", errBucketCollectFailed, connID, saved.Status)
	}
	if saved.CSPData == "" {
		return nil, errBucketNotCollected
	}
	var p cspBucketRecord
	if err := json.Unmarshal([]byte(saved.CSPData), &p); err != nil {
		return nil, errors.New("error occurred while parsing bucket information (ConnectionID = " + connID + ")")
	}
	return &p, nil
}

// isCSPBucketRecord reports whether the data record was written by a CSP
// bucket collection: it holds a CSP bucket, or it belongs to an object-storage
// connection whose first collection failed before any bucket was stored.
func isCSPBucketRecord(connID string, saved *model.SavedDataInfo) bool {
	if saved.CSPData != "" {
		return true
	}
	conn, err := dao.ConnectionInfoGet(connID)
	if err != nil || conn == nil {
		return false
	}

	return conn.ResourceType == serverCommon.ResourceTypeObjectStorage
}

// storageSourceCloud names the cloud the buckets come from. bucketRegion, the
// region the buckets were found in, wins over the group's region; it is empty
// where the CSP could not tell.
func storageSourceCloud(sg *model.SourceGroup, bucketRegion string) *storagemodel.CloudProperty {
	if !serverCommon.IsCSPType(sg.Type) {
		return nil
	}
	if bucketRegion != "" {
		return &storagemodel.CloudProperty{Csp: strings.ToLower(sg.ProviderName), Region: bucketRegion}
	}
	region, _, err := splitRegionZone(sg.RegionName, "")
	if err != nil {
		return nil
	}
	return &storagemodel.CloudProperty{Csp: strings.ToLower(sg.ProviderName), Region: region}
}

// groupBuckets picks the buckets the group's refined model can hold. The
// model names one source region for all of them, the group's, so a bucket
// found in another region is left out rather than labelled with the wrong one.
// Registration and import refuse such a bucket now; a record stored before
// they did may still hold one. A group without a region has none to compare
// with and keeps every bucket.
func groupBuckets(sg *model.SourceGroup, records []cspBucketRecord) []storagemodel.SourceObjectStorageProperty {
	groupRegion, _, regionErr := splitRegionZone(sg.RegionName, "")
	buckets := []storagemodel.SourceObjectStorageProperty{}
	for _, r := range records {
		if regionErr != nil {
			buckets = append(buckets, r.SourceObjectStorageProperty)
			continue
		}
		if err := bucketRegionMismatch(r.BucketName, r.Region, groupRegion); err != nil {
			logger.Println(logger.WARN, false, "group refined leaves a bucket out: "+err.Error())
			continue
		}
		buckets = append(buckets, r.SourceObjectStorageProperty)
	}

	return buckets
}

// GetDataInfoRefined godoc
//
//	@ID				get-data-info-refined
//	@Summary		Get Refined Object Storage Information
//	@Description	Get the bucket of a CSP object-storage connection in the source model cm-beetle recommends a target bucket from. The values are those collected by import/infra. sourceCloud.region is the region the bucket was found in, or the source group's region where the CSP cannot tell.
//	@Tags			[Get] Get refined source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Param			connId path string true "ID of the connection info."
//	@Success		200	{object}	storagemodel.SourceObjectStorageModel	"Successfully get refined information of the bucket."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get refined information of the bucket."
//	@Router			/source_group/{sgId}/connection_info/{connId}/data/refined [get]
func GetDataInfoRefined(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}
	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}
	sourceGroup, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	p, err := doGetRefinedBucket(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var m storagemodel.SourceObjectStorageModel
	m.SourceObjectStorageModel.SourceCloud = storageSourceCloud(sourceGroup, p.Region)
	m.SourceObjectStorageModel.SourceObjectStorages = []storagemodel.SourceObjectStorageProperty{p.SourceObjectStorageProperty}
	return c.JSONPretty(http.StatusOK, m, " ")
}

// GetDataInfoSourceGroupRefined godoc
//
//	@ID				get-data-info-source-group-refined
//	@Summary		Get Refined Object Storage Information Source Group
//	@Description	Get every collected bucket of the source group's CSP object-storage connections in the source model cm-beetle recommends target buckets from. Connections without a successfully collected bucket are left out, as are buckets found in a region other than the group's. sourceCloud.region is the group's region.
//	@Tags			[Get] Get refined source info
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the source group."
//	@Success		200	{object}	storagemodel.SourceObjectStorageModel	"Successfully get refined information of the buckets."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get refined information of the buckets."
//	@Router			/source_group/{sgId}/data/refined [get]
func GetDataInfoSourceGroupRefined(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}
	sourceGroup, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	list, err := dao.ConnectionInfoGetList(&model.ConnectionInfo{SourceGroupID: sgID}, 0, 0)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	records := []cspBucketRecord{}
	for _, conn := range *list {
		p, err := doGetRefinedBucket(conn.ID)
		if err != nil {
			if errors.Is(err, errBucketNotCollected) || errors.Is(err, errBucketCollectFailed) {
				continue
			}
			return common.ReturnErrorMsg(c, err.Error())
		}
		records = append(records, *p)
	}

	var m storagemodel.SourceObjectStorageModel
	m.SourceObjectStorageModel.SourceCloud = storageSourceCloud(sourceGroup, "")
	m.SourceObjectStorageModel.SourceObjectStorages = groupBuckets(sourceGroup, records)
	return c.JSONPretty(http.StatusOK, m, " ")
}
