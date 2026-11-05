package route

import (
	"strings"

	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
)

func RegisterImportObjectStorage(e *echo.Echo) {
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/connection_info/:connId/import/objectstorage", controller.ImportObjectStorage)
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/import/objectstorage", controller.ImportObjectStorageSourceGroup)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/connection_info/:connId/objectstorage", controller.GetObjectStorageInfo)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/objectstorage", controller.GetObjectStorageInfoSourceGroup)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/connection_info/:connId/objectstorage/refined", controller.GetObjectStorageInfoRefined)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/objectstorage/refined", controller.GetObjectStorageInfoSourceGroupRefined)
}
