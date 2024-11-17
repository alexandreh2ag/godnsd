package provider

import (
	"bytes"
	"encoding/json"
	"github.com/alexandreh2ag/go-dns-discover/config"
	"github.com/alexandreh2ag/go-dns-discover/context"
	"github.com/alexandreh2ag/go-dns-discover/types"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPI_GetId(t *testing.T) {
	id := "foo"
	a := &API{
		id: id,
	}
	assert.Equalf(t, id, a.GetId(), "GetId()")
}

func TestAPI_GetType(t *testing.T) {

	a := &API{}
	assert.Equalf(t, ApiKeyType, a.GetType(), "GetType()")
}

func Test_createApiProvider(t *testing.T) {
	ctx := context.TestContext(nil)
	got, err := createApiProvider(ctx, "foo", config.Provider{Type: "api"})
	assert.NoError(t, err)
	assert.NotNil(t, got)
}

func TestAPI_addRecord(t *testing.T) {
	ctx := context.TestContext(nil)

	tests := []struct {
		name    string
		record  *types.Record
		records types.Records
		want    types.Records
	}{
		{
			name:    "SuccessEmptyRecords",
			record:  &types.Record{Name: "foo.local", Type: "A", Value: "127.0.0.1"},
			records: types.Records{},
			want: types.Records{
				"foo.local._A": {{Name: "foo.local", Type: "A", Value: "127.0.0.1"}},
			},
		},
		{
			name:    "SuccessAppendRecords",
			record:  &types.Record{Name: "foo.local", Type: "A", Value: "127.0.0.2"},
			records: types.Records{"foo.local._A": {{Name: "foo.local", Type: "A", Value: "127.0.0.1"}}},
			want: types.Records{
				"foo.local._A": {
					{Name: "foo.local", Type: "A", Value: "127.0.0.1"},
					{Name: "foo.local", Type: "A", Value: "127.0.0.2"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &API{
				id:      "api",
				records: tt.records,
				logger:  ctx.Logger,
				notify:  make(chan bool),
				done:    ctx.Done(),
			}
			go a.addRecord(tt.record)
			<-a.notify
			assert.Equal(t, tt.want, a.records)
		})
	}
}

func TestAPI_deleteRecord(t *testing.T) {
	ctx := context.TestContext(nil)

	tests := []struct {
		name    string
		record  *types.Record
		records types.Records
		want    types.Records
	}{
		{
			name:   "SuccessEmptyRecords",
			record: &types.Record{Name: "foo.local", Type: "A", Value: "127.0.0.1"},
			records: types.Records{
				"foo.local._A": {{Name: "foo.local", Type: "A", Value: "127.0.0.1"}},
			},
			want: types.Records{},
		},
		{
			name:   "SuccessAppendRecords",
			record: &types.Record{Name: "foo.local", Type: "A", Value: "127.0.0.2"},
			want:   types.Records{"foo.local._A": {{Name: "foo.local", Type: "A", Value: "127.0.0.1"}}},
			records: types.Records{
				"foo.local._A": {
					{Name: "foo.local", Type: "A", Value: "127.0.0.1"},
					{Name: "foo.local", Type: "A", Value: "127.0.0.2"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &API{
				id:      "api",
				records: tt.records,
				logger:  ctx.Logger,
				notify:  make(chan bool),
				done:    ctx.Done(),
			}
			go a.deleteRecord(tt.record)
			<-a.notify
			assert.Equal(t, tt.want, a.records)
		})
	}
}

func TestAPI_Provide(t *testing.T) {
	ctx := context.TestContext(nil)
	records := types.Records{
		"foo.local._A": {{Name: "foo.local", Type: "A", Value: "127.0.0.1"}},
	}
	a := &API{
		id: "api",
		records: types.Records{
			"foo.local._A": {{Name: "foo.local", Type: "A", Value: "127.0.0.1"}},
		},
		logger: ctx.Logger,
		notify: make(chan bool),
		done:   ctx.Done(),
	}
	configurationChan := make(chan types.Message, 1)
	go func() {
		err := a.Provide(configurationChan)
		assert.NoError(t, err)
	}()
	a.notify <- true
	got := <-configurationChan
	assert.Equal(t, records, got.Records)
	ctx.Cancel()
}

func TestAPI_HandlerAddRecord(t *testing.T) {
	ctx := context.TestContext(nil)

	tests := []struct {
		name         string
		waitChan     bool
		records      types.Records
		body         interface{}
		wantHttpCode int
		wantErr      assert.ErrorAssertionFunc
	}{
		{
			name:         "Success",
			waitChan:     true,
			records:      types.Records{},
			body:         types.Record{Name: "foo.local", Type: "A", Value: "127.0.0.1"},
			wantHttpCode: http.StatusCreated,
			wantErr:      assert.NoError,
		},
		{
			name:         "ErrorParseBody",
			records:      types.Records{},
			body:         []string{"test"},
			wantHttpCode: http.StatusBadRequest,
			wantErr:      assert.NoError,
		},
		{
			name:         "ErrorParseBody",
			records:      types.Records{},
			body:         types.Record{},
			wantHttpCode: http.StatusBadRequest,
			wantErr:      assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &API{
				id:      "api",
				records: tt.records,
				logger:  ctx.Logger,
				notify:  make(chan bool),
				done:    ctx.Done(),
			}
			jsonBody, _ := json.Marshal(tt.body)
			body := bytes.NewReader(jsonBody)
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/", body)
			req.Header.Add("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			if tt.waitChan {
				go func() {
					tt.wantErr(t, a.HandlerAddRecord(c))
				}()
				<-a.notify
			} else {
				tt.wantErr(t, a.HandlerAddRecord(c))
			}
			assert.Equal(t, tt.wantHttpCode, rec.Code)
		})
	}
}

func TestAPI_HandlerDeleteRecord(t *testing.T) {
	ctx := context.TestContext(nil)

	tests := []struct {
		name         string
		waitChan     bool
		records      types.Records
		body         interface{}
		wantHttpCode int
		wantErr      assert.ErrorAssertionFunc
	}{
		{
			name:         "Success",
			waitChan:     true,
			records:      types.Records{},
			body:         types.Record{Name: "foo.local", Type: "A", Value: "127.0.0.1"},
			wantHttpCode: http.StatusOK,
			wantErr:      assert.NoError,
		},
		{
			name:         "ErrorParseBody",
			records:      types.Records{},
			body:         []string{"test"},
			wantHttpCode: http.StatusBadRequest,
			wantErr:      assert.NoError,
		},
		{
			name:         "ErrorParseBody",
			records:      types.Records{},
			body:         types.Record{},
			wantHttpCode: http.StatusBadRequest,
			wantErr:      assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &API{
				id:      "api",
				records: tt.records,
				logger:  ctx.Logger,
				notify:  make(chan bool),
				done:    ctx.Done(),
			}
			jsonBody, _ := json.Marshal(tt.body)
			body := bytes.NewReader(jsonBody)
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/", body)
			req.Header.Add("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			if tt.waitChan {
				go func() {
					tt.wantErr(t, a.HandlerDeleteRecord(c))
				}()
				<-a.notify
			} else {
				tt.wantErr(t, a.HandlerDeleteRecord(c))
			}
			assert.Equal(t, tt.wantHttpCode, rec.Code)
		})
	}
}

func TestAPI_HandlerPresent(t *testing.T) {
	ctx := context.TestContext(nil)

	tests := []struct {
		name         string
		waitChan     bool
		records      types.Records
		body         interface{}
		wantHttpCode int
		wantErr      assert.ErrorAssertionFunc
	}{
		{
			name:         "Success",
			waitChan:     true,
			records:      types.Records{},
			body:         httpRequestAcme{FQDN: "foo.local", Value: "127.0.0.1"},
			wantHttpCode: http.StatusCreated,
			wantErr:      assert.NoError,
		},
		{
			name:         "ErrorParseBody",
			records:      types.Records{},
			body:         []string{"test"},
			wantHttpCode: http.StatusBadRequest,
			wantErr:      assert.NoError,
		},
		{
			name:         "ErrorParseBody",
			records:      types.Records{},
			body:         httpRequestAcme{},
			wantHttpCode: http.StatusBadRequest,
			wantErr:      assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &API{
				id:      "api",
				records: tt.records,
				logger:  ctx.Logger,
				notify:  make(chan bool),
				done:    ctx.Done(),
			}
			jsonBody, _ := json.Marshal(tt.body)
			body := bytes.NewReader(jsonBody)
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/", body)
			req.Header.Add("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			if tt.waitChan {
				go func() {
					tt.wantErr(t, a.HandlerPresent(c))
				}()
				<-a.notify
			} else {
				tt.wantErr(t, a.HandlerPresent(c))
			}
			assert.Equal(t, tt.wantHttpCode, rec.Code)
		})
	}
}

func TestAPI_HandlerCleanup(t *testing.T) {
	ctx := context.TestContext(nil)

	tests := []struct {
		name         string
		waitChan     bool
		records      types.Records
		body         interface{}
		wantHttpCode int
		wantErr      assert.ErrorAssertionFunc
	}{
		{
			name:         "Success",
			waitChan:     true,
			records:      types.Records{},
			body:         httpRequestAcme{FQDN: "foo.local", Value: "127.0.0.1"},
			wantHttpCode: http.StatusOK,
			wantErr:      assert.NoError,
		},
		{
			name:         "ErrorParseBody",
			records:      types.Records{},
			body:         []string{"test"},
			wantHttpCode: http.StatusBadRequest,
			wantErr:      assert.NoError,
		},
		{
			name:         "ErrorParseBody",
			records:      types.Records{},
			body:         httpRequestAcme{},
			wantHttpCode: http.StatusBadRequest,
			wantErr:      assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &API{
				id:      "api",
				records: tt.records,
				logger:  ctx.Logger,
				notify:  make(chan bool),
				done:    ctx.Done(),
			}
			jsonBody, _ := json.Marshal(tt.body)
			body := bytes.NewReader(jsonBody)
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/", body)
			req.Header.Add("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			if tt.waitChan {
				go func() {
					tt.wantErr(t, a.HandlerCleanup(c))
				}()
				<-a.notify
			} else {
				tt.wantErr(t, a.HandlerCleanup(c))
			}
			assert.Equal(t, tt.wantHttpCode, rec.Code)
		})
	}
}
