package dns

import (
	"dario.cat/mergo"
	"fmt"
	"github.com/alexandreh2ag/go-dns-discover/context"
	"github.com/alexandreh2ag/go-dns-discover/types"
	"github.com/miekg/dns"
	"log/slog"
	"sync"
)

func CreateManager(ctx *context.Context, providers types.Providers) *Manager {
	return &Manager{logger: ctx.Logger, providers: providers, done: ctx.Done()}
}

type Manager struct {
	logger                *slog.Logger
	providers             types.Providers
	records               types.Records
	cacheProvidersRecords map[string]types.Records

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
				m.logger.Error(fmt.Sprintf("error when provide %s: %v", provider.GetId(), err))
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

func (m *Manager) HandleDnsRequest() func(w dns.ResponseWriter, r *dns.Msg) {
	return func(w dns.ResponseWriter, r *dns.Msg) {
		message := new(dns.Msg)
		message.SetReply(r)
		message.Compress = false

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
	key := types.FormatRecordKey(question.Name, types.ConvertTypeDNSUintToStr(question.Qtype))
	if entriesDns, ok := m.records[key]; ok {
		for _, record := range entriesDns {
			rr, err := dns.NewRR(
				fmt.Sprintf("%s %s %s", record.Name, record.Type, record.Value),
			)
			if err == nil {
				message.Answer = append(message.Answer, rr)
			}
		}
		return
	}

	if question.Qtype == dns.TypeA {
		keyCNAME := types.FormatRecordKey(question.Name, types.ConvertTypeDNSUintToStr(dns.TypeCNAME))
		if entriesDns, ok := m.records[keyCNAME]; ok {
			if len(entriesDns) == 0 {
				m.logger.Error(fmt.Sprintf("no DNS records for %s type CNAME", question.Name))
				return
			}
			record := entriesDns[0]
			rr, err := dns.NewRR(
				fmt.Sprintf("%s %s %s", record.Name, record.Type, record.Value),
			)
			if err == nil {
				message.Answer = append(message.Answer, rr)
			}
			m.answerQuestion(message, dns.Question{Name: record.Value, Qtype: dns.TypeA})
			return
		}
	}
}
