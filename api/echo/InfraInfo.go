package echo

import "github.com/labstack/echo/v4"

func GetInfraInfo(c echo.Context) error {
	c.QueryParam("")

	return nil
}

func InfraInfo() {
	e.GET("/infra", GetInfraInfo)
}
