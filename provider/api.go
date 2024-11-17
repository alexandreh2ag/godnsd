package provider

import (
	"fmt"
	"github.com/alexandreh2ag/go-dns-discover/config"
	"github.com/alexandreh2ag/go-dns-discover/context"
	"github.com/alexandreh2ag/go-dns-discover/types"
	"github.com/labstack/echo/v4"
	"log/slog"
	"net/http"
	"slices"
	"sync"
)

func init() {
	FactoryProviderMapping[ApiKeyType] = createApiProvider
}

const (
	ApiKeyType = "api"
)

var (
	_ types.Provider = &API{}
)

type httpRequestAcme struct {
	FQDN  string `json:"fqdn"`
	Value string `json:"value"`
}

type API struct {
	id      string
	logger  *slog.Logger
	records types.Records
	notify  chan bool
	done    chan bool
	mtx     sync.Mutex
}

func (a *API) GetId() string {
	return a.id
}

func (a *API) GetType() string {
	return ApiKeyType
}

func (a *API) Provide(configurationChan chan<- types.Message) error {

	for {
		select {
		case <-a.notify:
			configurationChan <- types.Message{Provider: a, Records: a.records}

		case <-a.done:
			return nil
		}
	}
}

func (a *API) HandlerAddRecord(c echo.Context) error {
	record := &types.Record{}
	if err := c.Bind(&record); err != nil {
		a.logger.Error(fmt.Sprintf("not recive record data: %s", err.Error()), "provider-type", a.GetType(), "provider-id", a.GetId())
		return c.NoContent(http.StatusBadRequest)
	}

	if record.Name == "" || record.Type == "" || record.Value == "" {
		a.logger.Error(fmt.Sprintf("record not valid: %v", record), "provider-type", a.GetType(), "provider-id", a.GetId())
		return c.NoContent(http.StatusBadRequest)
	}
	a.addRecord(record)

	return c.NoContent(http.StatusCreated)
}

func (a *API) HandlerPresent(c echo.Context) error {
	requestAcme := &httpRequestAcme{}

	if err := c.Bind(&requestAcme); err != nil {
		a.logger.Error(fmt.Sprintf("not recive record data: %s", err.Error()), "provider-type", a.GetType(), "provider-id", a.GetId())
		return c.NoContent(http.StatusBadRequest)
	}
	record := &types.Record{Name: requestAcme.FQDN, Type: "TXT", Value: requestAcme.Value}
	if record.Name == "" || record.Type == "" || record.Value == "" {
		a.logger.Error(fmt.Sprintf("record not valid: %v", record), "provider-type", a.GetType(), "provider-id", a.GetId())
		return c.NoContent(http.StatusBadRequest)
	}
	a.addRecord(record)

	return c.NoContent(http.StatusCreated)
}

func (a *API) HandlerDeleteRecord(c echo.Context) error {
	record := &types.Record{}
	if err := c.Bind(&record); err != nil {
		a.logger.Error(fmt.Sprintf("not recive record data: %s", err.Error()), "provider-type", a.GetType(), "provider-id", a.GetId())
		return c.NoContent(http.StatusBadRequest)
	}

	if record.Name == "" || record.Type == "" || record.Value == "" {
		a.logger.Error(fmt.Sprintf("record not valid: %v", record), "provider-type", a.GetType(), "provider-id", a.GetId())
		return c.NoContent(http.StatusBadRequest)
	}
	a.deleteRecord(record)
	return c.NoContent(http.StatusOK)
}

func (a *API) HandlerCleanup(c echo.Context) error {
	requestAcme := &httpRequestAcme{}

	if err := c.Bind(&requestAcme); err != nil {
		a.logger.Error(fmt.Sprintf("not recive record data: %s", err.Error()), "provider-type", a.GetType(), "provider-id", a.GetId())
		return c.NoContent(http.StatusBadRequest)
	}
	record := &types.Record{Name: requestAcme.FQDN, Type: "TXT", Value: requestAcme.Value}
	if record.Name == "" || record.Type == "" || record.Value == "" {
		a.logger.Error(fmt.Sprintf("record not valid: %v", record), "provider-type", a.GetType(), "provider-id", a.GetId())
		return c.NoContent(http.StatusBadRequest)
	}
	a.deleteRecord(record)

	return c.NoContent(http.StatusOK)
}

func (a *API) addRecord(record *types.Record) {
	key := types.FormatRecordKey(record.Name, record.Type)
	a.mtx.Lock()
	defer a.mtx.Unlock()
	if _, ok := a.records[key]; !ok {
		a.records[key] = []*types.Record{}
	}
	a.records[key] = append(a.records[key], record)
	a.notify <- true
}

func (a *API) deleteRecord(record *types.Record) {
	key := types.FormatRecordKey(record.Name, record.Type)
	a.mtx.Lock()
	defer a.mtx.Unlock()
	if _, ok := a.records[key]; ok {
		a.records[key] = slices.DeleteFunc(a.records[key], func(r *types.Record) bool {
			if r.Value == record.Value {
				return true
			}
			return false
		})
		if len(a.records[key]) == 0 {
			delete(a.records, key)
		}
	}
	a.notify <- true
}

func createApiProvider(ctx *context.Context, id string, cfg config.Provider) (types.Provider, error) {

	instance := &API{
		id:      id,
		notify:  make(chan bool),
		done:    ctx.Done(),
		records: types.Records{},
	}
	instance.logger = ctx.Logger
	return instance, nil
}
