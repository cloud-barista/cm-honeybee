package route

import (
	"strings"

	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
)

func RegisterImportFS(e *echo.Echo) {
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/connection_info/:connId/import/fs", controller.ImportFS)
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/import/fs", controller.ImportFSSourceGroup)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/connection_info/:connId/fs", controller.GetFSInfo)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/fs", controller.GetFSInfoSourceGroup)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/connection_info/:connId/fs/refined", controller.GetFSInfoRefined)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/fs/refined", controller.GetFSInfoSourceGroupRefined)
}
