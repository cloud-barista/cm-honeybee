package controller

import (
	"github.com/cloud-barista/cm-honeybee/agent/driver/kubernetes"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/common"
	_ "github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/kubernetes" // Need for swag
	"github.com/labstack/echo/v4"
	"net/http"
)

// GetKubernetesInfo godoc
//
//	@ID				get-kubernetes-info
//	@Summary		Get a list of integrated kubernetes information
//	@Description	Get kubernetes information.
//	@Tags			[Kubernetes] Get kubernetes info
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	kubernetes.Kubernetes	"Successfully get information of the kubernetes."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get information of the kubernetes."
//	@Router			/kubernetes [get]
func GetKubernetesInfo(c echo.Context) error {
	kubernetesInfo, err := infra.GetKubernetesInfo()
	if err != nil {
		return common.ReturnInternalError(c, err, "Failed to get information of the kubernetes.")
	}

	return c.JSONPretty(http.StatusOK, kubernetesInfo, " ")
}
