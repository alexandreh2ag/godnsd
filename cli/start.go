package cli

import (
	"fmt"
	"github.com/alexandreh2ag/go-dns-discover/context"
	appDns "github.com/alexandreh2ag/go-dns-discover/dns"
	"github.com/alexandreh2ag/go-dns-discover/provider"
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
