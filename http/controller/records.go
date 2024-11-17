package controller

import (
	"github.com/alexandreh2ag/go-dns-discover/dns"
	"github.com/labstack/echo/v4"
	"net/http"
)

func GetRecords(manager *dns.Manager) func(c echo.Context) error {
	return func(c echo.Context) error {
		records := manager.GetRecords()
		return c.JSON(http.StatusOK, records)
	}
}
