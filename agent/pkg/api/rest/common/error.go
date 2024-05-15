package common

import (
	"github.com/jollaman999/utils/logger"
	"github.com/labstack/echo/v4"
	"net/http"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func ReturnErrorMsg(c echo.Context, msg string) error {
	return c.JSONPretty(http.StatusBadRequest, ErrorResponse{Error: msg}, " ")
}

func ReturnInternalError(c echo.Context, err error, reason string) error {
	logger.Println(logger.ERROR, true, err.Error())

	msg := "Internal error occurred. (Reason: " + reason + ", Error: " + err.Error() + ")"

	return c.JSONPretty(http.StatusInternalServerError, ErrorResponse{Error: msg}, " ")
}
