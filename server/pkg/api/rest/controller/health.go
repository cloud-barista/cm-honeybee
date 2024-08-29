package controller

import (
	_ "github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/common" // Need for swag
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/labstack/echo/v4"
	"net/http"
)

var OkMessage = model.SimpleMsg{}
var IsReady = false

// CheckReady func is for checking Honeybee server health.
// @Summary Check Ready
// @Description Check Honeybee is ready
// @Tags [Admin] System management
// @Accept		json
// @Produce		json
// @Success		200 {object}	model.SimpleMsg			"Successfully get ready state."
// @Failure		500	{object}	common.ErrorResponse	"Failed to check ready state."
// @Router		/readyz [get]
func CheckReady(c echo.Context) error {
	status := http.StatusOK

	if !IsReady {
		status = http.StatusServiceUnavailable
	}

	return c.JSONPretty(status, &OkMessage, " ")
}
