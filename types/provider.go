package types

type Providers map[string]Provider
type Provider interface {
	GetId() string
	GetType() string
	Provide(configurationChan chan<- Message) error
}
