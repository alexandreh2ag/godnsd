package types

import (
	"github.com/miekg/dns"
	"time"
)

type ClientDNS interface {
	Exchange(m *dns.Msg, address string) (r *dns.Msg, rtt time.Duration, err error)
}
