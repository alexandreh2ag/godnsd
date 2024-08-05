package cli

import (
	"errors"
	"fmt"
	"github.com/alexandreh2ag/go-dns-discover/context"
	"github.com/alexandreh2ag/go-dns-discover/types"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log/slog"
	"path"
	"path/filepath"
)

const (
	AppName  = types.AppName
	Config   = "config"
	LogLevel = "level"
)

func GetRootCmd(ctx *context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:               AppName,
		Short:             "DNS server who discover records from providers",
		PersistentPreRunE: GetRootPreRunEFn(ctx),
	}

	cmd.PersistentFlags().StringP(Config, "c", "", "Define config path")
	cmd.PersistentFlags().StringP(LogLevel, "l", "INFO", "Define log level")
	_ = viper.BindPFlag(Config, cmd.Flags().Lookup(Config))
	_ = viper.BindPFlag(LogLevel, cmd.Flags().Lookup(LogLevel))
	viper.SetDefault(LogLevel, "info")
	viper.RegisterAlias("log_level", LogLevel)

	cmd.AddCommand(
		GetStartCmd(ctx),
		GetVersionCmd(),
	)
	return cmd
}

func GetRootPreRunEFn(ctx *context.Context) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		initConfig(ctx, cmd)

		logLevelFlagStr, _ := cmd.Flags().GetString(LogLevel)
		if logLevelFlagStr != "" && cmd.Flags().Changed(LogLevel) {
			level := slog.LevelInfo
			err := level.UnmarshalText([]byte(logLevelFlagStr))
			if err != nil {
				return err
			}
			ctx.LogLevel.Set(level)
		}
		ctx.Logger.Info(fmt.Sprintf("Log level %s", ctx.LogLevel.String()))
		validate := validator.New(validator.WithRequiredStructEnabled())
		err := validate.Struct(ctx.Config)
		if err != nil {
			var validationErrors validator.ValidationErrors
			switch {
			case errors.As(err, &validationErrors):
				for _, validationError := range validationErrors {
					ctx.Logger.Error(fmt.Sprintf("%v", validationError))
				}
				return errors.New("configuration file is not valid")
			default:
				return err
			}
		}

		return nil
	}
}

func initConfig(ctx *context.Context, cmd *cobra.Command) {

	viper.AddConfigPath(filepath.Join("/etc", AppName))
	viper.AutomaticEnv()
	viper.SetEnvPrefix(AppName)
	viper.SetConfigName(Config)
	viper.SetConfigType("yaml")

	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		panic(err)
	}

	configPath := viper.GetString(Config)

	if configPath != "" {
		viper.SetConfigFile(configPath)
		configDir := path.Dir(configPath)
		if configDir != "." {
			viper.AddConfigPath(configDir)
		}
	}
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		fmt.Println(err)
	}

	err := viper.Unmarshal(ctx.Config)
	if err != nil {
		panic(fmt.Errorf("unable to decode into config struct, %v", err))
	}

}
