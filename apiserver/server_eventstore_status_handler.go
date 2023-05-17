package apiserver

import (
	"github.com/alphagov/paas-billing/eventio"
	"github.com/labstack/echo/v4"
	"net/http"
)

func EventStoreStatusHandler(store eventio.EventStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		success := true
		status := http.StatusOK

		if err := store.Ping(); err != nil {
			success = false
			status = http.StatusInternalServerError
		}

		return c.JSONPretty(status, map[string]bool{
			"ok": success,
		}, "  ")
	}
}
