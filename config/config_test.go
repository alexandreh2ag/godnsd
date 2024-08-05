package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewConfig(t *testing.T) {
	got := NewConfig()

	assert.Equal(t, Config{}, got)
}

func TestDefaultConfig(t *testing.T) {
	got := DefaultConfig()
	want := Config{ListenAddr: "0.0.0.0:53", Providers: map[string]Provider{}}
	assert.Equal(t, want, got)
}
