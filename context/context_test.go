package context

import (
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/alexandreh2ag/go-dns-discover/config"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestNewContext(t *testing.T) {
	cfg := &config.Config{}
	level := &slog.LevelVar{}
	level.Set(slog.LevelInfo)
	opts := &slog.HandlerOptions{AddSource: false, Level: level}
	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
	fs := afero.NewMemMapFs()
	want := &Context{
		Config:   cfg,
		Logger:   logger,
		LogLevel: level,
		FS:       fs,
	}
	got := NewContext(cfg, logger, level, fs)
	got.done = nil
	got.sigs = nil
	assert.Equal(t, want, got)
}

func TestDefaultContext(t *testing.T) {
	level := &slog.LevelVar{}
	level.Set(slog.LevelInfo)
	opts := &slog.HandlerOptions{AddSource: false, Level: level}
	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
	cfg := config.DefaultConfig()
	want := &Context{
		Config:   &cfg,
		FS:       afero.NewOsFs(),
		Logger:   logger,
		LogLevel: level,
	}
	got := DefaultContext()
	got.done = nil
	got.sigs = nil
	assert.Equal(t, want, got)
}

func TestTestContext(t *testing.T) {

	cfg := config.DefaultConfig()
	level := &slog.LevelVar{}
	level.Set(slog.LevelInfo)
	opts := &slog.HandlerOptions{AddSource: false, Level: level}
	logger := slog.New(slog.NewTextHandler(io.Discard, opts))
	fs := afero.NewMemMapFs()
	want := &Context{
		Config:   &cfg,
		Logger:   logger,
		LogLevel: level,
		FS:       fs,
	}
	got := TestContext(nil)
	got.done = nil
	got.sigs = nil
	assert.Equal(t, want, got)
}

func TestContext_Sigs(t *testing.T) {
	sigs := make(chan os.Signal, 1)
	ctx := &Context{sigs: sigs}
	assert.Equal(t, sigs, ctx.Signal())
}

func TestContext_Done(t *testing.T) {
	done := make(chan bool)
	ctx := &Context{done: done}
	assert.Equal(t, done, ctx.Done())
}

func TestContext_Cancel(t *testing.T) {
	done := make(chan bool)
	ctx := &Context{done: done}
	go ctx.Cancel()
	got := <-done
	assert.True(t, got)
}
