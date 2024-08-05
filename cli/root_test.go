package cli

import (
	"bytes"
	"fmt"
	"github.com/alexandreh2ag/go-dns-discover/config"
	"github.com/alexandreh2ag/go-dns-discover/context"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"io"
	"path/filepath"
	"testing"
)

var (
	defaultConfigPath = filepath.Join("/etc", AppName)
)

func Test_initConfig_SuccessConfigEmpty(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetRootCmd(ctx)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	fsFake := afero.NewMemMapFs()
	viper.Reset()
	viper.SetFs(fsFake)

	_ = fsFake.Mkdir(defaultConfigPath, 0775)
	_ = afero.WriteFile(fsFake, fmt.Sprintf("%s/config.yml", defaultConfigPath), []byte(""), 0644)
	want := config.DefaultConfig()
	initConfig(ctx, cmd)
	assert.Equal(t, &want, ctx.Config)
}

func Test_initConfig_SuccessOverrideDefaultConfig(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetRootCmd(ctx)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	fsFake := afero.NewMemMapFs()
	viper.Reset()
	viper.SetFs(fsFake)
	_ = fsFake.Mkdir(defaultConfigPath, 0775)
	_ = afero.WriteFile(fsFake, fmt.Sprintf("%s/config.yml", defaultConfigPath), []byte(""), 0644)
	want := config.DefaultConfig()
	initConfig(ctx, cmd)
	assert.Equal(t, &want, ctx.Config)
}

func Test_initConfig_SuccessWithConfigFlag(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetRootCmd(ctx)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	fsFake := afero.NewMemMapFs()
	viper.Reset()
	viper.SetFs(fsFake)
	path := "/foo"
	_ = fsFake.Mkdir(path, 0775)
	_ = afero.WriteFile(fsFake, fmt.Sprintf("%s/foo.yml", path), []byte("listen_addr: 127.0.0.1:53"), 0644)
	want := config.DefaultConfig()
	want.ListenAddr = "127.0.0.1:53"
	viper.Set(Config, fmt.Sprintf("%s/foo.yml", path))
	initConfig(ctx, cmd)
	assert.Equal(t, &want, ctx.Config)
}

func Test_initConfig_FailWithUnmarshallError(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetRootCmd(ctx)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	fsFake := afero.NewMemMapFs()
	viper.Reset()
	viper.SetFs(fsFake)
	_ = fsFake.Mkdir(defaultConfigPath, 0775)
	_ = afero.WriteFile(fsFake, fmt.Sprintf("%s/config.yml", defaultConfigPath), []byte("listen_addr: ['test']"), 0644)
	assert.Panics(t, func() {
		initConfig(ctx, cmd)
	})
}

func TestGetRootPreRunEFn_Success(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetRootCmd(ctx)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	viper.Reset()
	viper.SetFs(ctx.FS)
	err := GetRootPreRunEFn(ctx)(cmd, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "LevelVar(INFO)", ctx.LogLevel.String())
}

func TestGetRootPreRunEFn_SuccessWithLogLevelFlag(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetRootCmd(ctx)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	fsFake := ctx.FS
	viper.Reset()
	viper.SetFs(fsFake)
	_ = fsFake.Mkdir(defaultConfigPath, 0775)
	_ = afero.WriteFile(fsFake, fmt.Sprintf("%s/config.yml", defaultConfigPath), []byte(""), 0644)
	cmd.SetArgs([]string{
		"--" + LogLevel, "ERROR"},
	)
	_ = cmd.Execute()

	err := GetRootPreRunEFn(ctx)(cmd, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "LevelVar(ERROR)", ctx.LogLevel.String())
}

func TestGetRootPreRunEFn_FailedWithLogLevelFlag(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetRootCmd(ctx)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	fsFake := afero.NewMemMapFs()
	viper.Reset()
	viper.SetFs(fsFake)
	_ = fsFake.Mkdir(defaultConfigPath, 0775)
	_ = afero.WriteFile(fsFake, fmt.Sprintf("%s/config.yml", defaultConfigPath), []byte(""), 0644)
	cmd.SetArgs([]string{
		"--" + LogLevel, "WRONG"},
	)
	_ = cmd.Execute()

	err := GetRootPreRunEFn(ctx)(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "slog: level string \"WRONG\": unknown name")
}

func TestGetRootPreRunEFn_FailedConfigValidator(t *testing.T) {
	b := bytes.NewBufferString("")
	ctx := context.TestContext(b)
	cmd := GetRootCmd(ctx)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	fsFake := afero.NewMemMapFs()
	viper.Reset()
	viper.SetFs(fsFake)
	_ = fsFake.Mkdir(defaultConfigPath, 0775)
	_ = afero.WriteFile(fsFake, fmt.Sprintf("%s/config.yml", defaultConfigPath), []byte("listen_addr: ''"), 0644)

	cmd.SetArgs([]string{})
	_ = cmd.Execute()

	err := GetRootPreRunEFn(ctx)(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "configuration file is not valid")
	assert.Contains(t, b.String(), "Key: 'Config.ListenAddr' Error:Field validation for 'ListenAddr' failed on the 'required' tag")
}
