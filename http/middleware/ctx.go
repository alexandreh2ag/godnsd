package middleware

import (
	"github.com/alexandreh2ag/go-dns-discover/context"
	"github.com/labstack/echo/v4"
)

const ContextKey = "context"

func HandlerContext(ctx *context.Context) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(ContextKey, ctx)
			return next(c)
		}
	}
}
