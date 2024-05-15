package common

import (
	"github.com/labstack/echo/v4"
	"strconv"
)

func CheckPageRow(c echo.Context) (page int, row int, err error) {
	pageS := c.QueryParam("page")
	if len(pageS) != 0 {
		page, err = strconv.Atoi(pageS)
		if err != nil || page < 0 {
			return -1, -1, ReturnErrorMsg(c, "Wrong page value.")
		}
	}

	rowS := c.QueryParam("row")
	if len(rowS) != 0 {
		row, err = strconv.Atoi(rowS)
		if err != nil || row < 0 {
			return -1, -1, ReturnErrorMsg(c, "Wrong row value.")
		}
	}

	return page, row, nil
}
