package api

import (
	"errors"
	"time"

	"github.com/labstack/echo"
)

// ValidateRangeParams validates and sets "from" and "to" params
func ValidateRangeParams(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := setRangeParams(c); err != nil {
			return err
		}
		return next(c)
	}
}

type RangeParams struct {
	From string
	To   string
}

func (params *RangeParams) Valid() error {
	_, err := time.Parse(time.RFC3339, params.From)
	if err != nil {
		return errors.New("invalid range param: 'from' must be RFC3339 format")
	}
	_, err = time.Parse(time.RFC3339, params.To)
	if err != nil {
		return errors.New("invalid range param: 'to' must be RFC3339 format")
	}
	return nil
}

func setRangeParams(c echo.Context) error {
	params := RangeParams{
		From: c.QueryParam("from"),
		To:   c.QueryParam("to"),
	}
	if params.From == "" {
		epoch := time.Time{}
		params.From = epoch.Format(time.RFC3339)
	}
	if params.To == "" {
		now := time.Now().UTC().Add(24 * time.Hour)
		params.To = now.Format(time.RFC3339)
	}
	if err := params.Valid(); err != nil {
		return err
	}
	c.Set("range", params)
	return nil
}
