package middleware

import (
	"github.com/alexandreh2ag/go-dns-discover/context"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerContext(t *testing.T) {
	ctx := context.TestContext(nil)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.Use(HandlerContext(ctx))

	h := func(c echo.Context) error {
		got := c.Get(ContextKey)
		assert.Equal(t, ctx, got)
		return c.NoContent(http.StatusOK)
	}
	e.GET("", h)
	e.ServeHTTP(rec, req)
}
