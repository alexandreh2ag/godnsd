package provider

import (
	"fmt"
	"github.com/alexandreh2ag/go-dns-discover/config"
	"github.com/alexandreh2ag/go-dns-discover/context"
	"github.com/alexandreh2ag/go-dns-discover/types"
)

var FactoryProviderMapping = map[string]CreateProviderFn{}

type CreateProviderFn func(ctx *context.Context, id string, cfg config.Provider) (types.Provider, error)

func CreateProviders(ctx *context.Context) (types.Providers, error) {
	instances := types.Providers{}
	for id, providerCfg := range ctx.Config.Providers {
		ctx.Logger.Debug(fmt.Sprintf("Create provider %s", id))
		instance, err := CreateProvider(ctx, id, providerCfg)
		if err != nil {
			return instances, err
		}
		instances[id] = instance
	}
	return instances, nil
}

func CreateProvider(ctx *context.Context, id string, cfg config.Provider) (types.Provider, error) {
	if fn, ok := FactoryProviderMapping[cfg.Type]; ok {
		return fn(ctx, id, cfg)
	}
	return nil, fmt.Errorf("provider type '%s' for %s does not exist", cfg.Type, id)
}
