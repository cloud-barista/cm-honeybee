package controller

import (
	"net/http"

	"github.com/cloud-barista/cm-honeybee/agent/driver/kubernetes"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/common"
	_ "github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/kubernetes" // Need for swag
	"github.com/labstack/echo/v4"
)

// GetHelmInfo godoc
//
//	@ID				get-helm-info
//	@Summary		Get a list of integrated helm information
//	@Description	Get helm information.
//	@Tags			[Kubernetes] Get helm info
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	kubernetes.Helm	"Successfully get information of the helm."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the helm."
//	@Router			/helm [get]
func GetHelmInfo(c echo.Context) error {
	helmInfo, err := kubernetes.GetHelmInfo()
	if err != nil {
		return common.ReturnInternalError(c, err, "Failed to get information of the helm.")
	}

	return c.JSONPretty(http.StatusOK, helmInfo, " ")
}
