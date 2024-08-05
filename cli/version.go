package cli

import (
	"github.com/spf13/cobra"

	"github.com/alexandreh2ag/go-dns-discover/version"
)

func GetVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version info",
		Run:   GetVersionRunFn(),
	}
}

func GetVersionRunFn() func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		cmd.Println(version.GetFormattedVersion())
	}
}
