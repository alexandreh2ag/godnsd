package types

type Message struct {
	Provider Provider
	Records  Records
}

func (m Message) GetProviderId() string {
	return m.Provider.GetId()
}
