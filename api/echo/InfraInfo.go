package echo

import (
	"github.com/cloud-barista/cm-honeybee/driver/infra"
	"github.com/labstack/echo/v4"
	"net/http"
)

func GetInfraInfo(c echo.Context) error {
	infraInfo, err := infra.GetInfraInfo()
	if err != nil {
		return returnInternalError(c, err, "Failed to get information of the infra.")
	}

	return c.JSONPretty(http.StatusOK, infraInfo, " ")
}

func InfraInfo() {
	e.GET("/infra", GetInfraInfo)
}
