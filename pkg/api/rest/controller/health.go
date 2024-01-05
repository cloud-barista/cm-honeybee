package controller

import (
	_ "github.com/cloud-barista/cm-honeybee/pkg/api/rest/common" // Need for swag
	"github.com/labstack/echo/v4"
	"net/http"
)

type SimpleMsg struct {
	Message string `json:"message"`
}

// GetHealth func is for checking Cicada server health.
// @Summary Check Cicada is alive
// @Description Check Cicada is alive
// @Tags [Admin] System management
// @Accept  json
// @Produce  json
// @Success		200 {object}	SimpleMsg	"Successfully get heath state."
// @Failure		500	{object}	common.ErrorResponse	"Failed to check health."
func GetHealth(c echo.Context) error {
	okMessage := SimpleMsg{}
	okMessage.Message = "CM-Honeybee API server is running"
	return c.JSONPretty(http.StatusOK, &okMessage, " ")
}
