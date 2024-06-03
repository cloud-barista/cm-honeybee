package route

import (
	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
	"strings"
)

func RegisterSourceGroup(e *echo.Echo) {
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/source_group", controller.CreateSourceGroup)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:uuid", controller.GetSourceGroup)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group", controller.ListSourceGroup)
	e.PUT("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:uuid", controller.UpdateSourceGroup)
	e.DELETE("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:uuid", controller.DeleteSourceGroup)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/check/:uuid", controller.CheckConnectionSourceGroup)
}
