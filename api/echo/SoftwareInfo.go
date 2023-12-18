package echo

import (
	"net/http"

	_ "github.com/cloud-barista/cm-honeybee/docs"
	"github.com/cloud-barista/cm-honeybee/driver/software"
	model "github.com/cloud-barista/cm-honeybee/model/software"
	"github.com/labstack/echo/v4"
)

type GetSoftwareResponse struct {
	// InfraList []infra.Infra `json:"infra"`
	model.Software
}

// GetSoftware godoc
//	@Summary		Get a list of Integrated Software information
//	@Description	Get information of all Software.
//	@Tags			[Sample] Software
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	GetSoftwareResponse	"(This is a sample description for success response in Swagger UI"
//	@Failure		404	{object}	GetSoftwareResponse	"Failed to get software"
//	@Router			/software [get]
func GetSoftwareInfo(c echo.Context) error {
	softwareInfo, err := software.GetSoftwareInfo()
	if err != nil {
		return returnInternalError(c, err, "Failed to get information of software.")
	}

	return c.JSONPretty(http.StatusOK, softwareInfo, " ")
}

func SoftwreInfo() {
	e.GET("/software", GetSoftwareInfo)
}
