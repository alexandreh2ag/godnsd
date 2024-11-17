package types

import (
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestRecord_UnmarshalYAML_Success(t *testing.T) {
	data := []byte("name: bar.foo.local\ntype: A\nvalue: 127.0.0.1")
	want := Record{Name: "bar.foo.local", Type: "A", Value: "127.0.0.1"}
	record := Record{}
	err := yaml.Unmarshal(data, &record)
	assert.NoError(t, err)
	assert.Equal(t, want, record)
}

func TestRecord_UnmarshalYAML_Failed(t *testing.T) {
	data := []byte("name: ['test']")
	record := Record{}
	err := yaml.Unmarshal(data, &record)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "yaml: unmarshal errors")
}

func TestRecords_UnmarshalYAML_Success(t *testing.T) {
	data := []byte("[{name: bar.foo.local,type: A, value: 127.0.0.1},{name: foo.foo.local,type: CNAME, value: bar.foo.local.},{name: bar.foo.local,type: A, value: 172.0.0.1}]")
	want := Records{"bar.foo.local._A": {{Name: "bar.foo.local", Type: "A", Value: "127.0.0.1"}, {Name: "bar.foo.local", Type: "A", Value: "172.0.0.1"}}, "foo.foo.local._CNAME": {{Name: "foo.foo.local", Type: "CNAME", Value: "bar.foo.local."}}}
	records := Records{}
	err := yaml.Unmarshal(data, &records)
	assert.NoError(t, err)
	recordsKeys := maps.Keys(records)
	wantKeys := maps.Keys(want)
	assert.ElementsMatch(t, wantKeys, recordsKeys)
}

func TestRecords_UnmarshalYAML_Failed(t *testing.T) {
	data := []byte("[{name: ['test']}]")
	records := Records{}
	err := yaml.Unmarshal(data, &records)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "yaml: unmarshal errors")
}

func Test_ConvertTypeDNSUintToStr(t *testing.T) {
	tests := []struct {
		name string
		Type uint16
		want string
	}{
		{
			name: "Type A",
			Type: dns.TypeA,
			want: "A",
		},
		{
			name: "Type AAAA",
			Type: dns.TypeAAAA,
			want: "AAAA",
		},
		{
			name: "Type CNAME",
			Type: dns.TypeCNAME,
			want: "CNAME",
		},
		{
			name: "Type TXT",
			Type: dns.TypeTXT,
			want: "TXT",
		},
		{
			name: "Type SOA",
			Type: dns.TypeSOA,
			want: "SOA",
		},
		{
			name: "Type NS",
			Type: dns.TypeNS,
			want: "NS",
		},
		{
			name: "Type Unknown",
			Type: 10000,
			want: "UNKNOWN",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, ConvertTypeDNSUintToStr(tt.Type), "ConvertTypeDNSUintToStr(%v)", tt.Type)
		})
	}
}
