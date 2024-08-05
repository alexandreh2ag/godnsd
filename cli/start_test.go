package cli

import (
	"bytes"
	"fmt"
	"github.com/alexandreh2ag/go-dns-discover/context"
	"github.com/miekg/dns"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"io"
	"syscall"
	"testing"
	"time"
)

func TestGetStartRunFn_Success(t *testing.T) {
	buffer := &bytes.Buffer{}
	ctx := context.TestContext(buffer)
	cmd := GetRootCmd(ctx)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	fsFake := ctx.FS
	viper.Reset()
	viper.SetFs(fsFake)
	path := "/app"
	_ = fsFake.Mkdir(path, 0775)

	_ = afero.WriteFile(fsFake, fmt.Sprintf("%s/config.yml", path), []byte("{listen_addr: '127.0.0.1:0', providers: {file: {type: fs, config: {path: /app/dns.yml}}}}"), 0644)
	_ = afero.WriteFile(fsFake, fmt.Sprintf("%s/dns.yml", path), []byte("[{name: foo.local, type: A, value: 127.0.0.1}]"), 0644)
	cmd.SetArgs([]string{CmdNameStart, "--" + Config, fmt.Sprintf("%s/config.yml", path)})
	go func() {
		err := cmd.Execute()
		assert.NoError(t, err)
	}()
	for server == nil || (server != nil && server.PacketConn.LocalAddr().String() == "") {
		time.Sleep(100 * time.Millisecond)
	}
	dnsClient := &dns.Client{Net: "udp", Timeout: 100 * time.Millisecond}
	req := &dns.Msg{
		MsgHdr:   dns.MsgHdr{Opcode: dns.OpcodeQuery},
		Question: []dns.Question{{Name: "foo.local.", Qtype: dns.TypeA, Qclass: dns.ClassINET}},
	}
	res, _, err := dnsClient.Exchange(req, server.PacketConn.LocalAddr().String())
	assert.NoError(t, err)
	assert.Contains(t, res.String(), "ANSWER SECTION:\nfoo.local.\t3600\tIN\tA\t127.0.0.1\n")

	ctx.Signal() <- syscall.SIGTERM
	time.Sleep(100 * time.Millisecond)
	assert.Contains(t, buffer.String(), "signal received, exiting...")
}

func TestGetStartRunFn_FailCreateProviders(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetRootCmd(ctx)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	fsFake := ctx.FS
	viper.Reset()
	viper.SetFs(fsFake)
	path := "/app"
	_ = fsFake.Mkdir(path, 0775)

	_ = afero.WriteFile(fsFake, fmt.Sprintf("%s/config.yml", path), []byte("{listen_addr: '127.0.0.1:0', providers: {file: {type: wrong}}}"), 0644)
	cmd.SetArgs([]string{CmdNameStart, "--" + Config, fmt.Sprintf("%s/config.yml", path)})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider type 'wrong' for file does not exist")
}

func TestGetStartRunFn_FailListen(t *testing.T) {
	buffer := &bytes.Buffer{}
	ctx := context.TestContext(buffer)
	cmd := GetRootCmd(ctx)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	fsFake := ctx.FS
	viper.Reset()
	viper.SetFs(fsFake)
	path := "/app"
	_ = fsFake.Mkdir(path, 0775)

	_ = afero.WriteFile(fsFake, fmt.Sprintf("%s/config.yml", path), []byte("{listen_addr: '127.0.0.1:-1', providers: {file: {type: fs, config: {path: /app/dns.yml}}}}"), 0644)
	cmd.SetArgs([]string{CmdNameStart, "--" + Config, fmt.Sprintf("%s/config.yml", path)})

	err := cmd.Execute()
	assert.Error(t, err)

}
