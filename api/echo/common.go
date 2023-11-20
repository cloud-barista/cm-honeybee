package echo

import (
	"github.com/jollaman999/utils/logger"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
)

func checkPageRow(c echo.Context) (page int, row int, err error) {
	pageS := c.QueryParam("page")
	if len(pageS) != 0 {
		page, err = strconv.Atoi(pageS)
		if err != nil || page < 0 {
			return -1, -1, returnErrorMsg(c, "Wrong page value.")
		}
	}

	rowS := c.QueryParam("row")
	if len(rowS) != 0 {
		row, err = strconv.Atoi(rowS)
		if err != nil || row < 0 {
			return -1, -1, returnErrorMsg(c, "Wrong row value.")
		}
	}

	return page, row, nil
}

func returnErrorMsg(c echo.Context, msg string) error {
	return c.JSONPretty(http.StatusBadRequest, map[string]string{
		"error": msg,
	}, " ")
}

func returnInternalError(c echo.Context, err error, reason string) error {
	logger.Println(logger.ERROR, true, err.Error())

	return returnErrorMsg(c, "Internal error occurred. (Reason: "+reason+", Error: "+err.Error()+")")
}
