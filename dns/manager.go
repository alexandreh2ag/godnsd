package dns

import (
	"dario.cat/mergo"
	"fmt"
	"github.com/alexandreh2ag/go-dns-discover/config"
	"github.com/alexandreh2ag/go-dns-discover/context"
	"github.com/alexandreh2ag/go-dns-discover/types"
	"github.com/miekg/dns"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"
)

func CreateManager(ctx *context.Context, providers types.Providers) *Manager {
	clientDNS := &dns.Client{Net: "udp", Timeout: time.Duration(ctx.Config.Fallback.Timeout) * time.Second}
	return &Manager{logger: ctx.Logger, providers: providers, done: ctx.Done(), fallbackCfg: ctx.Config.Fallback, clientDNS: clientDNS}
}

type Manager struct {
	logger                *slog.Logger
	fallbackCfg           config.FallbackConfig
	providers             types.Providers
	records               types.Records
	cacheProvidersRecords map[string]types.Records

	clientDNS         types.ClientDNS
	configurationChan chan types.Message
	done              chan bool
}

func (m *Manager) Start() {
	m.cacheProvidersRecords = make(map[string]types.Records)
	m.configurationChan = make(chan types.Message, 40)
	wg := sync.WaitGroup{}
	go m.listen()
	for _, provider := range m.providers {
		wg.Add(1)
		m.cacheProvidersRecords[provider.GetId()] = types.Records{}
		go func(prd types.Provider) {
			defer wg.Done()
			err := prd.Provide(m.configurationChan)
			if err != nil {
				m.logger.Error(fmt.Sprintf("error when provide %s: %v", prd.GetId(), err))
			}
		}(provider)
	}
	wg.Wait()
}

func (m *Manager) listen() {
	for {
		select {
		case message := <-m.configurationChan:
			if _, ok := m.cacheProvidersRecords[message.GetProviderId()]; !ok {
				m.logger.Error("routine received a message that does not belong to any provider")
				continue
			}
			m.logger.Debug(fmt.Sprintf("notification update config from %s with %d records", message.GetProviderId(), len(message.Records)))
			m.cacheProvidersRecords[message.GetProviderId()] = message.Records
			tmpRecords := types.Records{}

			for providerKey, providerRecords := range m.cacheProvidersRecords {
				err := mergo.Merge(&tmpRecords, providerRecords, mergo.WithAppendSlice)
				if err != nil {
					m.logger.Error(fmt.Sprintf("error when merging provider (%s) records: %v", providerKey, err))
				}
			}
			m.records = tmpRecords
		case <-m.done:
			close(m.configurationChan)
			return
		}
	}
}

func (m *Manager) GetRecords() types.Records {
	return m.records
}

func (m *Manager) HandleDnsRequest() func(w dns.ResponseWriter, r *dns.Msg) {
	return func(w dns.ResponseWriter, r *dns.Msg) {
		message := new(dns.Msg)
		message.SetReply(r)
		message.Compress = false
		m.logger.Debug(fmt.Sprintf("received a DNS message %s", message.String()))
		switch r.Opcode {
		case dns.OpcodeQuery:
			m.parseQuestions(message)
		}

		err := w.WriteMsg(message)
		if err != nil {
			m.logger.Error(fmt.Sprintf("error %v", err))
		}
	}
}

func (m *Manager) parseQuestions(message *dns.Msg) {
	for _, question := range message.Question {
		m.answerQuestion(message, question)
	}
}
func (m *Manager) answerQuestion(message *dns.Msg, question dns.Question) {
	records := m.findRecords(question)

	if len(records) > 0 {
		for _, record := range records {
			rr, err := dns.NewRR(
				fmt.Sprintf("%s %s %s", record.Name, record.Type, record.Value),
			)
			if err == nil {
				message.Answer = append(message.Answer, rr)
			}
		}
	} else {
		if m.fallbackCfg.Enable {
			msg := &dns.Msg{
				MsgHdr:   dns.MsgHdr{Id: message.Id, Opcode: dns.OpcodeQuery, RecursionDesired: true, RecursionAvailable: true},
				Question: []dns.Question{{Name: question.Name, Qtype: question.Qtype, Qclass: question.Qclass}},
			}
			for _, nameserver := range m.fallbackCfg.Nameservers {
				res, err := m.answerWithFallback(nameserver, msg)
				if err == nil {
					message.Answer = res.Answer
					return
				}
			}
		}
	}
}

func (m *Manager) findRecords(question dns.Question) []*types.Record {
	key := types.FormatRecordKey(question.Name, types.ConvertTypeDNSUintToStr(question.Qtype))
	if entriesDns, ok := m.records[key]; ok {
		return entriesDns
	}

	if question.Qtype == dns.TypeNS {
		domainSplit := strings.Split(question.Name, ".")
		if len(domainSplit) > 0 && question.Name != "*." {
			i := 1
			if domainSplit[0] == "*" {
				i = 2
			}
			recordsFound := m.findRecords(dns.Question{Name: "*." + strings.Join(domainSplit[i:len(domainSplit)], "."), Qtype: dns.TypeNS})

			for _, record := range recordsFound {
				record.Name = question.Name[:len(question.Name)-1]
			}
			return recordsFound
		}
	}

	if question.Qtype == dns.TypeA {
		keyCNAME := types.FormatRecordKey(question.Name, types.ConvertTypeDNSUintToStr(dns.TypeCNAME))
		if entriesDns, ok := m.records[keyCNAME]; ok {
			if len(entriesDns) == 0 {
				m.logger.Error(fmt.Sprintf("no DNS records for %s type CNAME", question.Name))
				return []*types.Record{}
			}
			record := entriesDns[0]
			records := m.findRecords(dns.Question{Name: record.Value, Qtype: dns.TypeA})
			return append([]*types.Record{record}, records...)
		}
	}

	if slices.Contains([]uint16{dns.TypeA, dns.TypeCNAME}, question.Qtype) && question.Name != "*." {
		domainSplit := strings.Split(question.Name, ".")
		if len(domainSplit) > 0 {
			i := 1
			if domainSplit[0] == "*" {
				i = 2
			}
			if question.Qtype == dns.TypeA {
				records := []*types.Record{}
				recordsFound := m.findRecords(dns.Question{Name: "*." + strings.Join(domainSplit[i:len(domainSplit)], "."), Qtype: dns.TypeA})

				for index, record := range recordsFound {
					name := record.Name
					typeDns := record.Type
					if index == 0 {
						name = question.Name[:len(question.Name)-1]
					}

					records = append(records, &types.Record{Name: name, Type: typeDns, Value: record.Value})
				}
				return records
			}
		}
	}
	return []*types.Record{}
}

func (m *Manager) answerWithFallback(nameserver string, message *dns.Msg) (*dns.Msg, error) {
	if i := strings.Index(nameserver, ":"); i < 0 {
		nameserver += ":53"
	}

	response, _, err := m.clientDNS.Exchange(message, nameserver)
	return response, err
}
