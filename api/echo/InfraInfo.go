package echo

import (
	"net/http"

	_ "github.com/cloud-barista/cm-honeybee/docs" // Honeybee Documentation
	"github.com/cloud-barista/cm-honeybee/driver/infra"
	model "github.com/cloud-barista/cm-honeybee/model/infra"
	"github.com/labstack/echo/v4"
)

type GetInfraResponse struct {
	// InfraList []infra.Infra `json:"infra"`
	model.Infra
}

// GetInfra godoc
//
//	@Summary		Get a list of Integrated Infra information
//	@Description	Get information of all Infra.
//	@Tags			[Sample] Infra
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	GetInfraResponse	"(This is a sample description for success response in Swagger UI"
//	@Failure		404	{object}	GetInfraResponse	"Failed to get infra"
//	@Router			/infra [get]
func GetInfraInfo(c echo.Context) error {
	infraInfo, err := infra.GetInfraInfo()
	if err != nil {
		return returnInternalError(c, err, "Failed to get information of the infra.")
	}

	return c.JSONPretty(http.StatusOK, infraInfo, " ")
}

func InfraInfo() {
	e.GET("/infra", GetInfraInfo)
}
