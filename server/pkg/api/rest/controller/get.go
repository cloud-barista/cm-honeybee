package controller

import (
	"encoding/json"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	_ "github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra" // Need for swag
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/labstack/echo/v4"
	"net/http"
)

// GetInfraInfo godoc
//
// @Summary		Get Infra Information
// @Description	Get the infra information of the connection information.
// @Tags		[Get] Get source info
// @Accept		json
// @Produce		json
// @Param		sgId path string true "ID of the source group."
// @Param		connId path string true "ID of the connection info."
// @Success		200	{object}	infra.Infra				"Successfully get information of the infra."
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to get information of the infra."
// @Router		/honeybee/source_group/{sgId}/connection_info/{connId}/infra [get]
func GetInfraInfo(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	savedInfraInfo, err := dao.SavedInfraInfoGet(connectionInfo.ID)
	if err != nil {
		return common.ReturnErrorMsg(c, "Failed to get information of the infra.")
	}
	var infraList infra.Infra
	err = json.Unmarshal([]byte(savedInfraInfo.InfraData), &infraList)
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while parsing software list.")
	}

	return c.JSONPretty(http.StatusOK, infraList, " ")
}

// GetSoftwareInfo godoc
//
// @Summary	Get Software Information
// @Description	Get the software information of the connection information.
// @Tags		[Get] Get source info
// @Accept		json
// @Produce	json
// @Param		sgId path string true "ID of the source group."
// @Param		connId path string true "ID of the connection info."
// @Success	200	{object}	software.Software		"Successfully get information of the software."
// @Failure	400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure	500	{object}	common.ErrorResponse	"Failed to get information of the software."
// @Router		/honeybee/source_group/{sgId}/connection_info/{connId}/software [get]
func GetSoftwareInfo(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}

	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
	}

	_, err := dao.SourceGroupGet(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	connectionInfo, err := dao.ConnectionInfoGet(connID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	savedSoftwareInfo, err := dao.SavedSoftwareInfoGet(connectionInfo.ID)
	if err != nil {
		return common.ReturnErrorMsg(c, "Failed to get information of the infra.")
	}
	var softwareList software.Software
	err = json.Unmarshal([]byte(savedSoftwareInfo.SoftwareData), &softwareList)
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while parsing software list.")
	}

	return c.JSONPretty(http.StatusOK, softwareList, " ")
}
