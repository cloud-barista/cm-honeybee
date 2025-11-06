package controller

import (
	"github.com/cloud-barista/cm-honeybee/agent/driver/data"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/common"
	_ "github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/data" // Need for swag
	"github.com/labstack/echo/v4"
	"net/http"
)

// GetDataInfo godoc
//
//	@ID				get-data-info
//	@Summary		Get data migration information
//	@Description	Get data migration information (required fields only for data migration).
//	@Tags			[Data] Get data migration info
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	data.DataInfo	"Successfully get data migration information."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get data migration information."
//	@Router			/data [get]
func GetDataInfo(c echo.Context) error {
	dataInfo, err := data.GetDataInfo()
	if err != nil {
		return common.ReturnInternalError(c, err, "Failed to get data migration information.")
	}

	return c.JSONPretty(http.StatusOK, dataInfo, " ")
}
