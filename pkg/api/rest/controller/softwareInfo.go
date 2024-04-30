package controller

import (
	"github.com/cloud-barista/cm-honeybee/driver/software"
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/common"
	_ "github.com/cloud-barista/cm-honeybee/pkg/api/rest/model/onprem/software" // Need for swag
	"github.com/labstack/echo/v4"
	"net/http"
)

// GetSoftwareInfo godoc
//
//	@Summary		Get a list of software information
//	@Description	Get software information.
//	@Tags			[Software] Get software info
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	software.Software	"Successfully get information of software."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of software."
//	@Router			/software [get]
func GetSoftwareInfo(c echo.Context) error {
	softwareInfo, err := software.GetSoftwareInfo()
	if err != nil {
		return common.ReturnInternalError(c, err, "Failed to get information of software.")
	}

	return c.JSONPretty(http.StatusOK, softwareInfo, " ")
}
