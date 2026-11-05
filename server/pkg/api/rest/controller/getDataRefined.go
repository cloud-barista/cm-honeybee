package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	storagemodel "github.com/cloud-barista/cm-beetle/imdl/storage-model"
	serverCommon "github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/labstack/echo/v4"
)

// errBucketNotCollected means the connection has no CSP bucket stored: it is
// not an object-storage connection of a CSP source group, or it was not
// imported yet.
var errBucketNotCollected = errors.New("no CSP bucket information was collected for this connection; run import/infra first")

func doGetRefinedBucket(connID string) (*storagemodel.SourceObjectStorageProperty, error) {
	saved, err := dao.SavedDataInfoGet(connID)
	if err != nil || saved == nil || saved.CSPData == "" {
		return nil, errBucketNotCollected
	}
	var p storagemodel.SourceObjectStorageProperty
	if err := json.Unmarshal([]byte(saved.CSPData), &p); err != nil {
		return nil, errors.New("error occurred while parsing bucket information (ConnectionID = " + connID + ")")
	}
	return &p, nil
}

func storageSourceCloud(sg *model.SourceGroup) *storagemodel.CloudProperty {
	if !serverCommon.IsCSPType(sg.Type) {
		return nil
	}
	region, _, err := splitRegionZone(sg.RegionName, "")
	if err != nil {
		return nil
	}
	return &storagemodel.CloudProperty{Csp: strings.ToLower(sg.ProviderName), Region: region}
}

// GetDataInfoRefined godoc
//
//	@ID				get-data-info-refined
//	@Summary		Get Refined Object Storage Information
//	@Description	Get the bucket of a CSP object-storage connection in the source model cm-beetle recommends a target bucket from. The values are those collected by import/infra.
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
	m.SourceObjectStorageModel.SourceCloud = storageSourceCloud(sourceGroup)
	m.SourceObjectStorageModel.SourceObjectStorages = []storagemodel.SourceObjectStorageProperty{*p}
	return c.JSONPretty(http.StatusOK, m, " ")
}

// GetDataInfoSourceGroupRefined godoc
//
//	@ID				get-data-info-source-group-refined
//	@Summary		Get Refined Object Storage Information Source Group
//	@Description	Get every collected bucket of the source group's CSP object-storage connections in the source model cm-beetle recommends target buckets from. Connections without a collected bucket are left out.
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

	buckets := []storagemodel.SourceObjectStorageProperty{}
	for _, conn := range *list {
		p, err := doGetRefinedBucket(conn.ID)
		if err != nil {
			if errors.Is(err, errBucketNotCollected) {
				continue
			}
			return common.ReturnErrorMsg(c, err.Error())
		}
		buckets = append(buckets, *p)
	}

	var m storagemodel.SourceObjectStorageModel
	m.SourceObjectStorageModel.SourceCloud = storageSourceCloud(sourceGroup)
	m.SourceObjectStorageModel.SourceObjectStorages = buckets
	return c.JSONPretty(http.StatusOK, m, " ")
}
