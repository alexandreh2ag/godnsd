package provider

import (
	"errors"
	"github.com/alexandreh2ag/go-dns-discover/config"
	"github.com/alexandreh2ag/go-dns-discover/context"
	mockTypes "github.com/alexandreh2ag/go-dns-discover/mocks/types"
	"github.com/alexandreh2ag/go-dns-discover/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"reflect"
	"testing"
)

func TestCreateProviders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	providerMock := mockTypes.NewMockProvider(ctrl)

	FactoryProviderMapping["dummy"] = func(ctx *context.Context, id string, cfg config.Provider) (types.Provider, error) {
		return providerMock, nil
	}

	FactoryProviderMapping["dummy_fail"] = func(ctx *context.Context, id string, cfg config.Provider) (types.Provider, error) {
		return nil, errors.New("fail")
	}

	tests := []struct {
		name         string
		want         types.Providers
		cfgProviders map[string]config.Provider
		wantErr      bool
	}{
		{
			name:         "SuccessNoProvider",
			cfgProviders: map[string]config.Provider{},
			want:         types.Providers{},
			wantErr:      false,
		},
		{
			name: "SuccessCreateProviders",
			cfgProviders: map[string]config.Provider{
				"foo": {Type: "dummy"},
				"bar": {Type: "dummy"},
			},
			want:    types.Providers{"foo": providerMock, "bar": providerMock},
			wantErr: false,
		},
		{
			name:         "FailedCreateProviders",
			cfgProviders: map[string]config.Provider{"foo": {Type: "dummy_fail"}},
			want:         types.Providers{},
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TestContext(nil)
			ctx.Config.Providers = tt.cfgProviders
			got, err := CreateProviders(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateProvider() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_CreateProvider_Success(t *testing.T) {
	ctx := context.TestContext(nil)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	providerMock := mockTypes.NewMockProvider(ctrl)

	FactoryProviderMapping["dummy"] = func(ctx *context.Context, id string, cfg config.Provider) (types.Provider, error) {
		return providerMock, nil
	}
	got, err := CreateProvider(ctx, "foo", config.Provider{Type: "dummy"})
	assert.NoError(t, err)
	assert.Equal(t, providerMock, got)
}

func Test_CreateProvider_FailedCreate(t *testing.T) {
	ctx := context.TestContext(nil)

	FactoryProviderMapping["dummy"] = func(ctx *context.Context, id string, cfg config.Provider) (types.Provider, error) {
		return nil, errors.New("fail")
	}
	got, err := CreateProvider(ctx, "foo", config.Provider{Type: "dummy"})
	assert.Error(t, err)
	assert.ErrorContains(t, err, "fail")
	assert.Nil(t, got)
}

func Test_CreateProvider_FailedUnknownType(t *testing.T) {
	ctx := context.TestContext(nil)

	got, err := CreateProvider(ctx, "foo", config.Provider{Type: "unknown"})
	assert.Error(t, err)
	assert.ErrorContains(t, err, "provider type 'unknown' for foo does not exist")
	assert.Nil(t, got)
}
