package controller

import (
	"encoding/json"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
	_ "github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software" // Need for swag
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/labstack/echo/v4"
	"net/http"
)

// GetSoftwareInfo godoc
//
//	@Summary		Get Software Information
//	@Description	Get the software information of the connection information.
//	@Tags			[Software] Get Software info
//	@Accept			json
//	@Produce		json
//	@Param			uuid path string true "ID of the connectionInfo"
//	@Success		200	{object}	software.Software		"Successfully get information of the software."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the software."
//	@Router			/honeybee/software/{connId} [get]
func GetSoftwareInfo(c echo.Context) error {
	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
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
