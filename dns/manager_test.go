package dns

import (
	"bytes"
	"errors"
	"github.com/alexandreh2ag/go-dns-discover/context"
	mockMiekgDns "github.com/alexandreh2ag/go-dns-discover/mocks/miekg"
	mockTypes "github.com/alexandreh2ag/go-dns-discover/mocks/types"
	"github.com/alexandreh2ag/go-dns-discover/types"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
	"time"
)

func TestCreateManager(t *testing.T) {
	ctx := context.TestContext(nil)
	providers := types.Providers{}
	got := CreateManager(ctx, providers)
	assert.NotNil(t, got)
}

func TestManager_listen(t *testing.T) {
	buffer := &bytes.Buffer{}
	ctx := context.TestContext(buffer)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	provider := mockTypes.NewMockProvider(ctrl)
	provider.EXPECT().GetId().AnyTimes().Return("provider")
	provider2 := mockTypes.NewMockProvider(ctrl)
	provider2.EXPECT().GetId().AnyTimes().Return("provider2")
	provider3 := mockTypes.NewMockProvider(ctrl)
	provider3.EXPECT().GetId().AnyTimes().Return("provider3")
	m := &Manager{
		logger:                ctx.Logger,
		cacheProvidersRecords: map[string]types.Records{"provider": {}, "provider2": {}},
		configurationChan:     make(chan types.Message, 40),
		done:                  ctx.Done(),
	}
	go m.listen()
	recordsPrd1 := types.Records{"foo.local._A": {{Name: "foo.local", Type: "A", Value: "127.0.0.1"}}}
	m.configurationChan <- types.Message{Provider: provider, Records: recordsPrd1}
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, recordsPrd1, m.records)
	assert.Equal(t, map[string]types.Records{"provider": recordsPrd1, "provider2": {}}, m.cacheProvidersRecords)

	recordsPrd2 := types.Records{
		"foo.local._A":     {{Name: "foo.local", Type: "A", Value: "127.0.0.2"}},
		"bar.local._CNAME": {{Name: "bar.local", Type: "CNAME", Value: "bar.local."}},
	}
	m.configurationChan <- types.Message{Provider: provider2, Records: recordsPrd2}
	time.Sleep(100 * time.Millisecond)

	assert.ElementsMatch(t, []*types.Record{{Name: "foo.local", Type: "A", Value: "127.0.0.1"}, {Name: "foo.local", Type: "A", Value: "127.0.0.2"}}, m.records["foo.local._A"])
	assert.ElementsMatch(t, []*types.Record{{Name: "bar.local", Type: "CNAME", Value: "bar.local."}}, m.records["bar.local._CNAME"])
	assert.Equal(t, map[string]types.Records{"provider": recordsPrd1, "provider2": recordsPrd2}, m.cacheProvidersRecords)

	m.configurationChan <- types.Message{Provider: provider2, Records: types.Records{}}
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, recordsPrd1, m.records)
	assert.Equal(t, map[string]types.Records{"provider": recordsPrd1, "provider2": {}}, m.cacheProvidersRecords)

	m.configurationChan <- types.Message{Provider: provider3, Records: types.Records{}}
	time.Sleep(100 * time.Millisecond)
	assert.Contains(t, buffer.String(), "routine received a message that does not belong to any provider")
	ctx.Cancel()
	<-m.configurationChan
}

func TestManager_Start_Success(t *testing.T) {
	ctx := context.TestContext(nil)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	provider := mockTypes.NewMockProvider(ctrl)
	provider.EXPECT().GetId().AnyTimes().Return("provider")
	provider.EXPECT().Provide(gomock.Any()).Times(1).Return(nil)
	m := &Manager{
		logger:    ctx.Logger,
		providers: types.Providers{"provider": provider},
		done:      make(chan bool),
	}
	m.Start()
	assert.Equal(t, map[string]types.Records{"provider": {}}, m.cacheProvidersRecords)
}
func TestManager_Start_Fail(t *testing.T) {
	buffer := &bytes.Buffer{}
	ctx := context.TestContext(buffer)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	provider := mockTypes.NewMockProvider(ctrl)
	provider.EXPECT().GetId().AnyTimes().Return("provider")
	provider.EXPECT().Provide(gomock.Any()).Times(1).Return(errors.New("fail"))
	m := &Manager{
		logger:    ctx.Logger,
		providers: types.Providers{"provider": provider},
		done:      make(chan bool),
	}
	m.Start()
	time.Sleep(100 * time.Millisecond)
	assert.Contains(t, buffer.String(), "fail")
	assert.Equal(t, map[string]types.Records{"provider": {}}, m.cacheProvidersRecords)
}

func TestManager_answerQuestion(t *testing.T) {
	ctx := context.TestContext(nil)

	tests := []struct {
		name    string
		records types.Records
		message *dns.Msg
		want    string
	}{
		{
			name:    "SuccessSimpleEntryA",
			records: types.Records{"foo.local._A": {{Name: "foo.local", Type: "A", Value: "127.0.0.1"}}},
			message: &dns.Msg{Question: []dns.Question{{Name: "foo.local.", Qtype: dns.TypeA, Qclass: dns.ClassINET}}},
			want:    "ANSWER SECTION:\nfoo.local.\t3600\tIN\tA\t127.0.0.1",
		},
		{
			name: "SuccessSimpleEntryCNAME",
			records: types.Records{
				"foo.local._A":         {{Name: "foo.local", Type: "A", Value: "127.0.0.1"}},
				"bar.foo.local._CNAME": {{Name: "bar.foo.local", Type: "CNAME", Value: "foo.local."}},
			},
			message: &dns.Msg{Question: []dns.Question{{Name: "bar.foo.local.", Qtype: dns.TypeA, Qclass: dns.ClassINET}}},
			want:    "ANSWER SECTION:\nbar.foo.local.\t3600\tIN\tCNAME\tfoo.local.\nfoo.local.\t3600\tIN\tA\t127.0.0.1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{
				logger:  ctx.Logger,
				records: tt.records,
			}
			m.answerQuestion(tt.message, tt.message.Question[0])
			assert.Contains(t, tt.message.String(), tt.want)
		})
	}
}

func TestManager_parseQuestions(t *testing.T) {
	ctx := context.TestContext(nil)

	records := types.Records{"foo.local._A": {{Name: "foo.local", Type: "A", Value: "127.0.0.1"}}, "bar.local._A": {{Name: "bar.local", Type: "A", Value: "127.0.0.1"}}}
	message := &dns.Msg{Question: []dns.Question{{Name: "foo.local.", Qtype: dns.TypeA, Qclass: dns.ClassINET}, {Name: "bar.local.", Qtype: dns.TypeA, Qclass: dns.ClassINET}}}
	want := "ANSWER SECTION:\nfoo.local.\t3600\tIN\tA\t127.0.0.1\nbar.local.\t3600\tIN\tA\t127.0.0.1\n"
	m := &Manager{
		logger:  ctx.Logger,
		records: records,
	}
	m.parseQuestions(message)
	assert.Contains(t, message.String(), want)
}

func TestManager_HandleDnsRequest_Success(t *testing.T) {
	buffer := &bytes.Buffer{}
	ctx := context.TestContext(buffer)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	records := types.Records{"foo.local._A": {{Name: "foo.local", Type: "A", Value: "127.0.0.1"}}}
	message := &dns.Msg{MsgHdr: dns.MsgHdr{Opcode: dns.OpcodeQuery}, Question: []dns.Question{{Name: "foo.local.", Qtype: dns.TypeA, Qclass: dns.ClassINET}}}
	want := "ANSWER SECTION:\nfoo.local.\t3600\tIN\tA\t127.0.0.1\n"
	m := &Manager{
		logger:  ctx.Logger,
		records: records,
	}
	responseWriter := mockMiekgDns.NewMockResponseWriter(ctrl)
	responseWriter.EXPECT().WriteMsg(gomock.Any()).DoAndReturn(func(msg *dns.Msg) error {
		assert.Contains(t, msg.String(), want)
		return nil
	})

	m.HandleDnsRequest()(responseWriter, message)
	assert.Empty(t, buffer.String())
}

func TestManager_HandleDnsRequest_Fail(t *testing.T) {
	buffer := &bytes.Buffer{}
	ctx := context.TestContext(buffer)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	records := types.Records{"foo.local._A": {{Name: "foo.local", Type: "A", Value: "127.0.0.1"}}}
	message := &dns.Msg{MsgHdr: dns.MsgHdr{Opcode: dns.OpcodeQuery}, Question: []dns.Question{{Name: "foo.local.", Qtype: dns.TypeA, Qclass: dns.ClassINET}}}
	m := &Manager{
		logger:  ctx.Logger,
		records: records,
	}
	responseWriter := mockMiekgDns.NewMockResponseWriter(ctrl)
	responseWriter.EXPECT().WriteMsg(gomock.Any()).DoAndReturn(func(msg *dns.Msg) error {
		return errors.New("fail")
	})

	m.HandleDnsRequest()(responseWriter, message)
	assert.Contains(t, buffer.String(), "fail")
}

func TestManager_findRecords(t *testing.T) {
	ctx := context.TestContext(nil)

	records := types.Records{
		"foo.local._A":             {{Name: "foo.local", Type: "A", Value: "127.0.0.1"}},
		"bar.foo.local._CNAME":     {{Name: "bar.foo.local", Type: "CNAME", Value: "foo.local."}},
		"wrong.foo.local._CNAME":   {},
		"*.foo.local._A":           {{Name: "*.foo.local", Type: "A", Value: "127.0.0.3"}},
		"*.other.local._CNAME":     {{Name: "*.foo.local", Type: "CNAME", Value: "foo.local."}},
		"*.foo.other.local._CNAME": {{Name: "*.foo.local", Type: "CNAME", Value: "wildcard.other.local."}},
		"*.test._A":                {{Name: "*.test", Type: "A", Value: "127.0.0.4"}},
	}

	tests := []struct {
		name     string
		records  types.Records
		question dns.Question
		want     []*types.Record
	}{
		{
			name:     "SuccessSimpleEntryA",
			records:  records,
			question: dns.Question{Name: "foo.local.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
			want:     append([]*types.Record{}, records["foo.local._A"]...),
		},
		{
			name:     "SuccessSimpleEntryCNAME",
			records:  records,
			question: dns.Question{Name: "bar.foo.local.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
			want:     []*types.Record{{Name: "bar.foo.local", Type: "CNAME", Value: "foo.local."}, {Name: "foo.local", Type: "A", Value: "127.0.0.1"}},
		},
		{
			name:     "SuccessWildcardWholeTld",
			records:  records,
			question: dns.Question{Name: "foo.test.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
			want:     []*types.Record{{Name: "foo.test", Type: "A", Value: "127.0.0.4"}},
		},
		{
			name:     "SuccessWildcardTypeAFirstParentDomain",
			records:  records,
			question: dns.Question{Name: "wildcard.foo.local.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
			want:     []*types.Record{{Name: "wildcard.foo.local", Type: "A", Value: "127.0.0.3"}},
		},
		{
			name:     "SuccessWildcardTypeASecondParentDomain",
			records:  records,
			question: dns.Question{Name: "wildcard.second.foo.local.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
			want:     []*types.Record{{Name: "wildcard.second.foo.local", Type: "A", Value: "127.0.0.3"}},
		},
		{
			name:     "SuccessWildcardTypeCnameFirstParentDomain",
			records:  records,
			question: dns.Question{Name: "wildcard.other.local.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
			want:     []*types.Record{{Name: "wildcard.other.local", Type: "CNAME", Value: "foo.local."}, {Name: "foo.local", Type: "A", Value: "127.0.0.1"}},
		},
		{
			name:     "SuccessWildcardTypeCnameFirstParentDomain",
			records:  records,
			question: dns.Question{Name: "wildcard.foo.other.local.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
			want:     []*types.Record{{Name: "wildcard.foo.other.local", Type: "CNAME", Value: "wildcard.other.local."}, {Name: "wildcard.other.local", Type: "CNAME", Value: "foo.local."}, {Name: "foo.local", Type: "A", Value: "127.0.0.1"}},
		},
		{
			name:     "SuccessEmptyCNAMEEntry",
			records:  records,
			question: dns.Question{Name: "wrong.foo.local.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
			want:     []*types.Record{},
		},
		{
			name:     "SuccessNoResult",
			records:  records,
			question: dns.Question{Name: "wrong.local.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
			want:     []*types.Record{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{
				logger:  ctx.Logger,
				records: tt.records,
			}
			got := m.findRecords(tt.question, true)
			assert.Equal(t, tt.want, got)
		})
	}
}
