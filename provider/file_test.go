package provider

import (
	"fmt"
	"github.com/alexandreh2ag/go-dns-discover/config"
	"github.com/alexandreh2ag/go-dns-discover/context"
	"github.com/alexandreh2ag/go-dns-discover/types"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_createFSProvider(t *testing.T) {
	ctx := context.TestContext(nil)

	tests := []struct {
		name    string
		cfg     config.Provider
		want    types.Provider
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "Success",
			cfg:     config.Provider{Config: map[string]interface{}{"path": "/app"}},
			want:    &FS{id: "provider", fs: ctx.FS, cfg: configFS{Path: "/app"}},
			wantErr: assert.NoError,
		},
		{
			name:    "FailDecodeCfg",
			cfg:     config.Provider{Config: map[string]interface{}{"path": []string{"wrong"}}},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name:    "FailValidate",
			cfg:     config.Provider{Config: map[string]interface{}{}},
			want:    nil,
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createFSProvider(ctx, "provider", tt.cfg)
			if !tt.wantErr(t, err, fmt.Sprintf("createFSProvider(%v, %v, %v)", ctx, "provider", tt.cfg)) {
				return
			}
			assert.Equalf(t, tt.want, got, "createFSProvider(%v, %v, %v)", ctx, "provider", tt.cfg)
		})
	}
}

func TestFS_readFile(t *testing.T) {

	tests := []struct {
		name    string
		mockFn  func(fs afero.Fs)
		want    types.Records
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Success",
			mockFn: func(fs afero.Fs) {
				_ = fs.Mkdir("/app", 0755)
				_ = afero.WriteFile(fs, "/app/config.yml", []byte("[{name: foo.bar.local, type: A, value: 127.0.0.1}]"), 0644)
			},
			want:    types.Records{"foo.bar.local._A": {{Name: "foo.bar.local", Type: "A", Value: "127.0.0.1"}}},
			wantErr: assert.NoError,
		},
		{
			name:    "FailOpenFile",
			mockFn:  func(fs afero.Fs) {},
			want:    types.Records{},
			wantErr: assert.Error,
		},
		{
			name: "FailMarshal",
			mockFn: func(fs afero.Fs) {
				_ = fs.Mkdir("/app", 0755)
				_ = afero.WriteFile(fs, "/app/config.yml", []byte("[}"), 0644)
			},
			want:    types.Records{},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			f := FS{
				id:  "provider",
				fs:  fs,
				cfg: configFS{Path: "/app/config.yml"},
			}
			if tt.mockFn != nil {
				tt.mockFn(fs)
			}
			got, err := f.readFile(f.cfg.Path)
			if !tt.wantErr(t, err, fmt.Sprintf("readFile(%v)", f.cfg.Path)) {
				return
			}
			assert.Equalf(t, tt.want, got, "readFile(%v)", f.cfg.Path)
		})
	}
}

func TestFS_GetId(t *testing.T) {
	f := FS{id: "provider"}
	assert.Equalf(t, "provider", f.GetId(), "GetId()")
}

func TestFS_GetType(t *testing.T) {
	f := FS{}
	assert.Equalf(t, fsKeyType, f.GetType(), "GetType()")
}

func TestFS_Provide(t *testing.T) {

	tests := []struct {
		name    string
		cfg     configFS
		mockFn  func(fs afero.Fs)
		ch      chan types.Message
		want    *types.Message
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "SuccessWithOneFile",
			cfg:  configFS{Path: "/app/config.yml"},
			ch:   make(chan types.Message, 1),
			mockFn: func(fs afero.Fs) {
				_ = fs.Mkdir("/app", 0755)
				_ = afero.WriteFile(fs, "/app/config.yml", []byte("[{name: foo.bar.local, type: A, value: 127.0.0.1},{name: foo.bar.local, type: A, value: 127.0.0.2},{name: bar.local, type: CNAME, value: foo.bar.local.}]"), 0644)
			},
			want: &types.Message{
				Records: types.Records{
					"foo.bar.local._A": {{Name: "foo.bar.local", Type: "A", Value: "127.0.0.1"}, {Name: "foo.bar.local", Type: "A", Value: "127.0.0.2"}},
					"bar.local._CNAME": {{Name: "bar.local", Type: "CNAME", Value: "foo.bar.local."}},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "SuccessWithDir",
			cfg:  configFS{Path: "/app"},
			ch:   make(chan types.Message, 1),
			mockFn: func(fs afero.Fs) {
				_ = fs.Mkdir("/app", 0755)
				_ = afero.WriteFile(fs, "/app/config.yml", []byte("[{name: foo.bar.local, type: A, value: 127.0.0.1},{name: foo.bar.local, type: A, value: 127.0.0.2},{name: bar.local, type: CNAME, value: foo.bar.local.}]"), 0644)
				_ = afero.WriteFile(fs, "/app/config2.yml", []byte("[{name: bar.local, type: CNAME, value: foo.bar.local.}]"), 0644)
			},
			want: &types.Message{
				Records: types.Records{
					"foo.bar.local._A": {{Name: "foo.bar.local", Type: "A", Value: "127.0.0.1"}, {Name: "foo.bar.local", Type: "A", Value: "127.0.0.2"}},
					"bar.local._CNAME": {{Name: "bar.local", Type: "CNAME", Value: "foo.bar.local."}, {Name: "bar.local", Type: "CNAME", Value: "foo.bar.local."}},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "FailOpenFile",
			cfg:  configFS{Path: "/app/config.yml"},
			ch:   make(chan types.Message, 1),
			mockFn: func(fs afero.Fs) {
				_ = fs.Mkdir("/app", 0755)
			},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			f := FS{
				id:  "provider",
				fs:  fs,
				cfg: tt.cfg,
			}
			if tt.want != nil {
				tt.want.Provider = f
			}
			if tt.mockFn != nil {
				tt.mockFn(fs)
			}
			go func() {
				tt.wantErr(t, f.Provide(tt.ch), fmt.Sprintf("Provide(chan)"))
			}()

			if tt.want != nil {
				got := <-tt.ch
				assert.Equal(t, *tt.want, got)
			}

		})
	}
}
