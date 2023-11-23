package echo

import (
	"github.com/cloud-barista/cm-honeybee/driver/software"
	"github.com/labstack/echo/v4"
	"net/http"
)

func GetSoftwareInfo(c echo.Context) error {
	softwareInfo, err := software.GetSoftwareInfo()
	if err != nil {
		return returnInternalError(c, err, "Failed to get information of software.")
	}

	return c.JSONPretty(http.StatusOK, softwareInfo, " ")
}

func SoftwreInfo() {
	e.GET("/software", GetSoftwareInfo)
}
