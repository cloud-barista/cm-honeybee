package route

import (
	"strings"

	"github.com/cloud-barista/cm-honeybee/agent/common"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/controller"
	_ "github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/docs" // Honeybee Documentation
	"github.com/labstack/echo/v4"
)

func RegisterDB(e *echo.Echo) {
	// POST only. A GET taking password and tls_ca_pem as query parameters would
	// leave them in this host's process list, in this agent's access log and in
	// the honeybee server's debug log; the body travels on the SSH session's
	// stdin instead and reaches none of the three.
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/db", controller.PostDBInfo)
}
