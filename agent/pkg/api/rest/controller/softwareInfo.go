package controller

import (
	"github.com/cloud-barista/cm-honeybee/agent/driver/software"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/common"
	_ "github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software" // Need for swag
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
)

// GetSoftwareInfo godoc
//
//	@ID				get-software-info
//	@Summary		Get a list of software information
//	@Description	Get software information.
//	@Tags			[Software] Get software info
//	@Accept			json
//	@Produce		json
//	@Param			show_default_packages query bool false "Enable for show all packages include default packages."
//	@Success		200	{object}	software.Software	"Successfully get information of software."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of software."
//	@Router			/software [get]
func GetSoftwareInfo(c echo.Context) error {
	showDefaultPackagesStr := c.QueryParam("show_default_packages")
	showDefaultPackages, _ := strconv.ParseBool(showDefaultPackagesStr)

	softwareInfo, err := software.GetSoftwareInfo(showDefaultPackages)
	if err != nil {
		return common.ReturnInternalError(c, err, "Failed to get information of software.")
	}

	return c.JSONPretty(http.StatusOK, softwareInfo, " ")
}
