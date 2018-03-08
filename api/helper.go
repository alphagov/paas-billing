package api

import (
	"fmt"
	"strconv"

	"github.com/labstack/echo"
)

func parseIntParam(c echo.Context, name string) (int, error) {
	valueStr := c.Param(name)
	if valueStr == "" {
		return 0, fmt.Errorf("%s is missing", name)
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, fmt.Errorf("%s is invalid, it must be a numeric value", name)
	}
	return value, nil
}
