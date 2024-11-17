package controller

import (
	"encoding/json"
	"github.com/alexandreh2ag/go-dns-discover/config"
	"github.com/alexandreh2ag/go-dns-discover/context"
	"github.com/alexandreh2ag/go-dns-discover/dns"
	"github.com/alexandreh2ag/go-dns-discover/http/middleware"
	"github.com/alexandreh2ag/go-dns-discover/provider"
	"github.com/alexandreh2ag/go-dns-discover/types"
	"github.com/labstack/echo/v4"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetRecords(t *testing.T) {
	records := types.Records{
		"foo.local._A": {{Name: "foo.local", Type: "A", Value: "127.0.0.1"}},
	}
	wantJson, _ := json.Marshal(&records)
	ctx := context.TestContext(nil)
	_ = afero.WriteFile(ctx.FS, "/app/config.yml", []byte("[{name: foo.local, type: A, value: 127.0.0.1}]"), 0644)
	p, errProvider := provider.CreateProvider(ctx, "fs", config.Provider{Type: "fs", Config: map[string]interface{}{"path": "/app/config.yml"}})
	assert.NoError(t, errProvider)
	m := dns.CreateManager(ctx, types.Providers{"fs": p})
	m.Start()
	time.Sleep(500 * time.Millisecond)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.ContextKey, ctx)
	err := GetRecords(m)(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, string(wantJson)+"\n", rec.Body.String())
}
