package route

import (
	"github.com/cloud-barista/cm-honeybee/server/common"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
	"strings"
)

func RegisterGetRefined(e *echo.Echo) {
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/connection_info/:connId/infra/refined", controller.GetInfraInfoRefined)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/infra/refined", controller.GetInfraInfoSourceGroupRefined)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/connection_info/:connId/software/refined", controller.GetSoftwareInfoRefined)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/software/refined", controller.GetSoftwareInfoSourceGroupRefined)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/connection_info/:connId/rdbms/refined", controller.GetRDBMSInfoRefined)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/rdbms/refined", controller.GetRDBMSInfoSourceGroupRefined)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/connection_info/:connId/data/refined", controller.GetDataInfoRefined)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/data/refined", controller.GetDataInfoSourceGroupRefined)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/connection_info/:connId/bucket/refined", controller.GetBucketInfoRefined)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/bucket/refined", controller.GetBucketInfoSourceGroupRefined)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/connection_info/:connId/k8s/refined", controller.GetK8sInfoRefined)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/k8s/refined", controller.GetK8sInfoSourceGroupRefined)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/connection_info/:connId/nlb/refined", controller.GetNLBInfoRefined)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/source_group/:sgId/nlb/refined", controller.GetNLBInfoSourceGroupRefined)
}
