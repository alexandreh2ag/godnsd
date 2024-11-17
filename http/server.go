package http

import "github.com/labstack/echo/v4"

func CreateEcho() *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	return e
}
