package types

import (
	"fmt"
	"github.com/miekg/dns"
	"gopkg.in/yaml.v3"
)

type Records map[string][]*Record

func (r Records) UnmarshalYAML(value *yaml.Node) error {
	tmp := []*Record{}
	if err := value.Decode(&tmp); err != nil {
		return err
	}

	for _, record := range tmp {
		key := FormatRecordKey(record.Name, record.Type)
		if _, ok := r[key]; !ok {
			r[key] = []*Record{}
		}
		r[key] = append(r[key], record)
	}
	return nil
}

type Record struct {
	Name  string `yaml:"name"`
	Type  string `yaml:"type"`
	Value string `yaml:"value"`
}

func FormatRecordKey(name string, typeRecord string) string {
	return fmt.Sprintf("%s_%s", dns.Fqdn(name), typeRecord)
}

func ConvertTypeDNSUintToStr(typeRecord uint16) string {
	switch typeRecord {
	case dns.TypeA:
		return "A"
	case dns.TypeAAAA:
		return "AAAA"
	case dns.TypeCNAME:
		return "CNAME"
	case dns.TypeTXT:
		return "TXT"
	default:
		return "A"
	}
}
