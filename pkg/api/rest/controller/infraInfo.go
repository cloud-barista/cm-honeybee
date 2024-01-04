package controller

import (
	"github.com/cloud-barista/cm-honeybee/driver/infra"
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/common"
	model "github.com/cloud-barista/cm-honeybee/pkg/api/rest/model/infra"
	"github.com/labstack/echo/v4"
	"net/http"
)

type GetInfraResponse struct {
	// InfraList []infra.Infra `json:"infra"`
	model.Infra
}

// GetInfraInfo godoc
//
//	@Summary		Get a list of Integrated Infra information
//	@Description	Get information of all Infra.
//	@Tags			[Sample] Infra
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	GetInfraResponse	"Successfully get information of the infra."
//	@Failure		404	{object}	GetInfraResponse	"Failed to get information of the infra."
//	@Router			/infra [get]
func GetInfraInfo(c echo.Context) error {
	infraInfo, err := infra.GetInfraInfo()
	if err != nil {
		return common.ReturnInternalError(c, err, "Failed to get information of the infra.")
	}

	return c.JSONPretty(http.StatusOK, infraInfo, " ")
}
