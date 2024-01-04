package controller

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

type SimpleMsg struct {
	Message string `json:"message" example:"Any message"`
}

// GetHealth func is for checking Honeybee server health.
// RestGetHealth godoc
// @Summary Check Honeybee is alive
// @Description Check Honeybee is alive
// @Tags [Admin] System management
// @Accept  json
// @Produce  json
// @Success 200 {object} SimpleMsg
// @Failure 404 {object} SimpleMsg
// @Failure 500 {object} SimpleMsg
// @Router /health [get]
func GetHealth(c echo.Context) error {
	okMessage := SimpleMsg{}
	okMessage.Message = "CM-Honeybee API server is running"
	return c.JSONPretty(http.StatusOK, &okMessage, " ")
}
