package route

import (
	"strings"

	"github.com/cloud-barista/cm-honeybee/agent/common"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/controller"
	_ "github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/docs" // Honeybee Documentation
	"github.com/labstack/echo/v4"
)

func RegisterFS(e *echo.Echo) {
	// POST only. The filesystem inspect carries no credentials, so there is no
	// secret a query string could leak here; the route stays POST to keep one
	// transport for all three inspects, which is what the object storage and
	// database ones require.
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/fs", controller.PostFSInfo)
}
