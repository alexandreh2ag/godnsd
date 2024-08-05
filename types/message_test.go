package types

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var _ Provider = &DummyProvider{}

type DummyProvider struct {
}

func (d DummyProvider) GetId() string {
	return "dummy"
}

func (d DummyProvider) GetType() string {
	return "dummy"
}

func (d DummyProvider) Provide(_ chan<- Message) error {
	return nil
}

func TestMessage_GetProviderId(t *testing.T) {
	m := Message{
		Provider: &DummyProvider{},
	}
	assert.Equal(t, "dummy", m.GetProviderId())
}
