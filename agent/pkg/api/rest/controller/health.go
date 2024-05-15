package controller

import (
	_ "github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/common" // Need for swag
	"github.com/labstack/echo/v4"
	"net/http"
)

type SimpleMsg struct {
	Message string `json:"message"`
}

// GetHealth func is for checking Honeybee Agent health.
// @Summary Check Honeybee Agent is alive
// @Description Check Honeybee Agent is alive
// @Tags [Admin] System management
// @Accept  json
// @Produce  json
// @Success		200 {object}	SimpleMsg	"Successfully get heath state."
// @Failure		500	{object}	common.ErrorResponse	"Failed to check health."
func GetHealth(c echo.Context) error {
	okMessage := SimpleMsg{}
	okMessage.Message = "CM-Honeybee Agent is running"
	return c.JSONPretty(http.StatusOK, &okMessage, " ")
}
