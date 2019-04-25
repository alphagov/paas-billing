package apiserver

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/lib/pq"
)

type ErrorResponse struct {
	Error      string `json:"error"`
	Constraint string `json:"constraint,omitempty"`
}

func errorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	resp := ErrorResponse{
		Error: "internal server error",
	}

	switch v := err.(type) {
	case *echo.HTTPError:
		code = v.Code
		resp.Error = fmt.Sprintf("%s", v.Message)
	case *pq.Error:
		if v.Code.Name() == "check_violation" {
			code = http.StatusBadRequest
			resp.Error = "constraint violation"
			resp.Constraint = v.Constraint
		}
	}

	c.Logger().Error(err)
	if err := c.JSON(code, resp); err != nil {
		c.Logger().Error(err)
	}
}
