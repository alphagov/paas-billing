package auth

import (
	"errors"
	"strings"

	"github.com/labstack/echo"
)

func GetTokenFromRequest(c echo.Context) (string, error) {
	if t := c.Request().Header.Get(echo.HeaderAuthorization); t != "" {
		parts := strings.Split(t, " ")
		if len(parts) != 2 {
			return "", errors.New("invalid Authorization header")
		}
		if strings.ToLower(parts[0]) != "bearer" {
			return "", errors.New("unsupported Authorization header type")
		}
		if parts[1] == "" {
			return "", errors.New("missing Authorization Bearer token data")
		}
		return parts[1], nil
	}
	return "", errors.New("no access_token in request")
}
