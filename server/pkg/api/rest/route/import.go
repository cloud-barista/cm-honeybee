package route

import (
	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
	"strings"
)

func RegisterImport(e *echo.Echo) {
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/connection_info/:connId/import/infra", controller.ImportInfra)
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/import/infra", controller.ImportInfraSourceGroup)
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/connection_info/:connId/import/software", controller.ImportSoftware)
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/import/software", controller.ImportSoftwareSourceGroup)
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/connection_info/:connId/import/kubernetes", controller.ImportKubernetes)
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/import/kubernetes", controller.ImportKubernetesSourceGroup)
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/connection_info/:connId/import/helm", controller.ImportHelm)
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/import/helm", controller.ImportHelmSourceGroup)
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/connection_info/:connId/import/data", controller.ImportData)
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/import/data", controller.ImportDataSourceGroup)
}
