package controller

import (
	"github.com/cloud-barista/cm-honeybee/driver/software"
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/common"
	model "github.com/cloud-barista/cm-honeybee/pkg/api/rest/model/software"
	"github.com/labstack/echo/v4"
	"net/http"
)

type GetSoftwareResponse struct {
	// InfraList []infra.Infra `json:"infra"`
	model.Software
}

// GetSoftwareInfo godoc
//
//	@Summary		Get a list of Integrated Software information
//	@Description	Get information of all Software.
//	@Tags			[Sample] Software
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	GetSoftwareResponse	"Successfully get information of software."
//	@Failure		404	{object}	GetSoftwareResponse	"Failed to get information of software."
//	@Router			/software [get]
func GetSoftwareInfo(c echo.Context) error {
	softwareInfo, err := software.GetSoftwareInfo()
	if err != nil {
		return common.ReturnInternalError(c, err, "Failed to get information of software.")
	}

	return c.JSONPretty(http.StatusOK, softwareInfo, " ")
}
