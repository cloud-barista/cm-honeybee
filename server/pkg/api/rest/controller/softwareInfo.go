package controller

import (
	"encoding/json"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
	_ "github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software" // Need for swag
	"github.com/cloud-barista/cm-honeybee/dao"
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/common"
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
//	@Param			uuid path string true "UUID of the connectionInfo"
//	@Success		200	{object}	software.Software		"Successfully get information of the software."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the software."
//	@Router			/software/{uuid} [get]
func GetSoftwareInfo(c echo.Context) error {
	uuid := c.Param("uuid")
	if uuid == "" {
		return common.ReturnErrorMsg(c, "uuid is empty")
	}

	connectionInfo, err := dao.ConnectionInfoGet(uuid)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	savedSoftwareInfo, err := dao.SavedSoftwareInfoGet(connectionInfo.UUID)
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
