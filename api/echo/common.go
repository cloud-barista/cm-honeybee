package echo

import (
	"github.com/jollaman999/utils/logger"
	"github.com/labstack/echo/v4"
	"net/http"
)

func returnErrorMsg(c echo.Context, msg string) error {
	return c.JSONPretty(http.StatusBadRequest, map[string]string{
		"error": msg,
	}, " ")
}

func returnInternalError(c echo.Context, err error, reason string) error {
	logger.Println(logger.ERROR, true, err.Error())

	return returnErrorMsg(c, "Internal error occurred. (Reason: "+reason+", Error: "+err.Error()+")")
}
