package config

type Config struct {
	ListenAddr string              `mapstructure:"listen_addr" validate:"required"`
	Providers  map[string]Provider `mapstructure:"providers" validate:"omitempty,required,dive"`
}

type Provider struct {
	Type   string                 `mapstructure:"type" validate:"required"`
	Config map[string]interface{} `mapstructure:"config"`
}

func NewConfig() Config {
	return Config{}
}

func DefaultConfig() Config {
	cfg := NewConfig()
	cfg.ListenAddr = "0.0.0.0:53"
	cfg.Providers = map[string]Provider{}
	return cfg
}
