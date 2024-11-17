package cli

import (
	"fmt"
	"github.com/alexandreh2ag/go-dns-discover/config"
	"github.com/alexandreh2ag/go-dns-discover/context"
	appDns "github.com/alexandreh2ag/go-dns-discover/dns"
	"github.com/alexandreh2ag/go-dns-discover/http"
	"github.com/alexandreh2ag/go-dns-discover/http/controller"
	"github.com/alexandreh2ag/go-dns-discover/http/middleware"
	"github.com/alexandreh2ag/go-dns-discover/provider"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/miekg/dns"
	"github.com/spf13/cobra"
)

const (
	CmdNameStart = "start"
)

var (
	server *dns.Server
)

func GetStartCmd(ctx *context.Context) *cobra.Command {
	return &cobra.Command{
		Use:   CmdNameStart,
		Short: "Start DNS server",
		RunE:  GetStartRunFn(ctx),
	}
}

func GetStartRunFn(ctx *context.Context) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {

		providers, err := provider.CreateProviders(ctx)
		if err != nil {
			return err
		}

		manager := appDns.CreateManager(ctx, providers)

		if ctx.Config.Http.Enable {
			e := http.CreateEcho()
			e.Use(
				echoMiddleware.GzipWithConfig(echoMiddleware.GzipConfig{
					Level: 5,
				}),
				middleware.HandlerContext(ctx),
			)
			apiGroup := e.Group("/api")
			apiRecordsGroup := apiGroup.Group("/records")
			apiRecordsGroup.GET("", controller.GetRecords(manager))

			if ctx.Config.Http.Enable && ctx.Config.Http.EnableApiProvider {
				apiId := "api"
				p, _ := provider.CreateProvider(ctx, apiId, config.Provider{Type: provider.ApiKeyType})
				providers[apiId] = p
				providerApi := p.(*provider.API)
				apiRecordsGroup.POST("", providerApi.HandlerAddRecord)
				apiRecordsGroup.DELETE("", providerApi.HandlerDeleteRecord)
				apiRecordsGroup.POST("/present", providerApi.HandlerPresent)
				apiRecordsGroup.POST("/cleanup", providerApi.HandlerCleanup)

			}

			go func() {
				errHttpStart := e.Start(ctx.Config.Http.Listen)
				ctx.Logger.Error(errHttpStart.Error())
			}()
		}

		go manager.Start()

		server = &dns.Server{Addr: ctx.Config.ListenAddr, Net: "udp"}
		dns.HandleFunc(".", manager.HandleDnsRequest())
		go func() {
			for {
				select {
				case sig := <-ctx.Signal():
					ctx.Cancel()
					ctx.Logger.Info(fmt.Sprintf("%s signal received, exiting...", sig.String()))
					err = server.Shutdown()
					if err != nil {
						ctx.Logger.Error(fmt.Sprintf("Failed to shutdown server: %s", err.Error()))
					}
				}
			}
		}()

		ctx.Logger.Info(fmt.Sprintf("Starting at %s", ctx.Config.ListenAddr))
		return server.ListenAndServe()
	}
}
