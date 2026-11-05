package route

import (
	"strings"

	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
)

func RegisterImportDB(e *echo.Echo) {
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/connection_info/:connId/import/db", controller.ImportDB)
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/import/db", controller.ImportDBSourceGroup)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/connection_info/:connId/db", controller.GetDBInfo)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/db", controller.GetDBInfoSourceGroup)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/connection_info/:connId/db/refined", controller.GetDBInfoRefined)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/db/refined", controller.GetDBInfoSourceGroupRefined)
}
