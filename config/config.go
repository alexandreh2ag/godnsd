package config

type Config struct {
	ListenAddr string              `mapstructure:"listen_addr" validate:"required"`
	Providers  map[string]Provider `mapstructure:"providers" validate:"omitempty,required,dive"`
	Fallback   FallbackConfig      `mapstructure:"fallback" validate:"omitempty,required"`
}

type Provider struct {
	Type   string                 `mapstructure:"type" validate:"required"`
	Config map[string]interface{} `mapstructure:"config"`
}

type FallbackConfig struct {
	Enable      bool     `mapstructure:"enable"`
	Nameservers []string `mapstructure:"nameservers" validate:"required_if=Enable true,dive,required"`
	Timeout     int64    `mapstructure:"timeout" validate:"omitempty,required"`
}

func NewConfig() Config {
	return Config{}
}

func DefaultConfig() Config {
	cfg := NewConfig()
	cfg.ListenAddr = "0.0.0.0:53"
	cfg.Providers = map[string]Provider{}
	cfg.Fallback.Timeout = 4
	return cfg
}
