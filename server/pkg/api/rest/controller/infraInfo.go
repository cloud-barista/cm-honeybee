package controller

import (
	"encoding/json"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	_ "github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra" // Need for swag
	"github.com/cloud-barista/cm-honeybee/server/dao"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common"
	"github.com/labstack/echo/v4"
	"net/http"
)

// GetInfraInfo godoc
//
//	@Summary		Get Infra Information
//	@Description	Get the infra information of the connection information.
//	@Tags			[Infra] Get infra info
//	@Accept			json
//	@Produce		json
//	@Param			uuid path string true "ID of the connectionInfo"
//	@Success		200	{object}	infra.Infra				"Successfully get information of the infra."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the infra."
//	@Router			/honeybee/infra/{connId} [get]
func GetInfraInfo(c echo.Context) error {
	connID := c.Param("connId")
	if connID == "" {
		return common.ReturnErrorMsg(c, "Please provide the connId.")
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
