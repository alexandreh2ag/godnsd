package context

import (
	"github.com/alexandreh2ag/go-dns-discover/config"
	"github.com/spf13/afero"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

type Context struct {
	Config   *config.Config
	Logger   *slog.Logger
	LogLevel *slog.LevelVar
	FS       afero.Fs

	done chan bool
	sigs chan os.Signal
}

func (c *Context) Signal() chan os.Signal {
	return c.sigs
}

func (c *Context) Cancel() {
	c.done <- true
}

func (c *Context) Done() chan bool {
	return c.done
}

func NewContext(config *config.Config, logger *slog.Logger, logLevel *slog.LevelVar, FSProvider afero.Fs) *Context {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	return &Context{Config: config, Logger: logger, LogLevel: logLevel, FS: FSProvider, sigs: sigs, done: make(chan bool)}
}

func DefaultContext() *Context {
	level := &slog.LevelVar{}
	level.Set(slog.LevelInfo)
	opts := &slog.HandlerOptions{AddSource: false, Level: level}

	cfg := config.DefaultConfig()
	return NewContext(&cfg, slog.New(slog.NewTextHandler(os.Stdout, opts)), level, afero.NewOsFs())
}

func TestContext(logBuffer io.Writer) *Context {
	if logBuffer == nil {
		logBuffer = io.Discard
	}
	cfg := config.DefaultConfig()
	level := &slog.LevelVar{}
	level.Set(slog.LevelInfo)
	opts := &slog.HandlerOptions{AddSource: false, Level: level}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	return &Context{
		Logger:   slog.New(slog.NewTextHandler(logBuffer, opts)),
		LogLevel: level,
		Config:   &cfg,
		FS:       afero.NewMemMapFs(),
		done:     make(chan bool),
		sigs:     sigs,
	}
}
