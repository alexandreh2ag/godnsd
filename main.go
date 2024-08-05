package main

import (
	"github.com/alexandreh2ag/go-dns-discover/cli"
	"github.com/alexandreh2ag/go-dns-discover/context"
)

func main() {
	ctx := context.DefaultContext()
	rootCmd := cli.GetRootCmd(ctx)

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
