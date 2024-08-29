package controller

import (
	"github.com/cloud-barista/cm-honeybee/agent/driver/infra"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/common"
	_ "github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra" // Need for swag
	"github.com/labstack/echo/v4"
	"net/http"
)

// GetInfraInfo godoc
//
//	@Summary		Get a list of integrated infra information
//	@Description	Get infra information.
//	@Tags			[Infra] Get infra info
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	infra.Infra	"Successfully get information of the infra."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the infra."
//	@Router			/infra [get]
func GetInfraInfo(c echo.Context) error {
	infraInfo, err := infra.GetInfraInfo()
	if err != nil {
		return common.ReturnInternalError(c, err, "Failed to get information of the infra.")
	}

	return c.JSONPretty(http.StatusOK, infraInfo, " ")
}
