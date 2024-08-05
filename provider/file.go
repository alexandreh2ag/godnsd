package provider

import (
	"dario.cat/mergo"
	"github.com/alexandreh2ag/go-dns-discover/config"
	"github.com/alexandreh2ag/go-dns-discover/context"
	"github.com/alexandreh2ag/go-dns-discover/types"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
	"os"
)

func init() {
	FactoryProviderMapping[fsKeyType] = createFSProvider
}

const (
	fsKeyType = "fs"
)

var (
	_ types.Provider = &FS{}
)

type configFS struct {
	Path string `mapstructure:"path" validate:"required"`
}

type FS struct {
	id  string
	fs  afero.Fs
	cfg configFS
}

func (f FS) GetId() string {
	return f.id
}

func (f FS) GetType() string {
	return fsKeyType
}

func (f FS) Provide(configurationChan chan<- types.Message) error {
	var err error

	records := types.Records{}
	if ok, _ := afero.IsDir(f.fs, f.cfg.Path); ok {
		err = afero.Walk(f.fs, f.cfg.Path, func(path string, info os.FileInfo, err error) error {
			recordsWalk, errWalk := f.readFile(path)
			if errWalk != nil {
				return errWalk
			}

			return mergo.Merge(&records, recordsWalk, mergo.WithAppendSlice)
		})
	} else {
		records, err = f.readFile(f.cfg.Path)
		if err != nil {
			return err
		}
	}

	configurationChan <- types.Message{Provider: f, Records: records}

	return nil
}

func (f FS) readFile(filename string) (types.Records, error) {
	records := types.Records{}
	content, err := afero.ReadFile(f.fs, filename)
	if err != nil {
		return records, err
	}

	err = yaml.Unmarshal(content, &records)
	if err != nil {
		return records, err
	}
	return records, nil
}

func createFSProvider(ctx *context.Context, id string, cfg config.Provider) (types.Provider, error) {
	instanceConfig := configFS{}
	err := mapstructure.Decode(cfg.Config, &instanceConfig)
	if err != nil {
		return nil, err
	}

	validate := validator.New()
	err = validate.Struct(instanceConfig)
	if err != nil {
		return nil, err
	}

	instance := &FS{
		id:  id,
		fs:  ctx.FS,
		cfg: instanceConfig,
	}
	return instance, nil
}
