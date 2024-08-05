package provider

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/alexandreh2ag/go-dns-discover/config"
	"github.com/alexandreh2ag/go-dns-discover/context"
	mockDocker "github.com/alexandreh2ag/go-dns-discover/mocks/docker"
	"github.com/alexandreh2ag/go-dns-discover/types"
	dockerTypes "github.com/docker/docker/api/types"
	dockerEvents "github.com/docker/docker/api/types/events"
	dockerNetwork "github.com/docker/docker/api/types/network"
	docketClient "github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
	"time"
)

func Test_createDockerProvider(t *testing.T) {
	ctx := context.TestContext(nil)
	ctrl := gomock.NewController(t)
	client := mockDocker.NewMockAPIClient(ctrl)
	tests := []struct {
		name           string
		createClientFn func() (docketClient.APIClient, error)
		want           types.Provider
		wantErr        assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			createClientFn: func() (docketClient.APIClient, error) {
				return client, nil
			},
			want:    &Docker{id: "provider", logger: ctx.Logger, client: client, done: ctx.Done()},
			wantErr: assert.NoError,
		},
		{
			name: "failCreateClientDocker",
			createClientFn: func() (docketClient.APIClient, error) {
				return nil, errors.New("fail to create client")
			},
			want:    nil,
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dockerClientFn = tt.createClientFn
			got, err := createDockerProvider(ctx, "provider", config.Provider{})
			if !tt.wantErr(t, err, fmt.Sprintf("createDockerProvider(ctx, 'provider', cfg)")) {
				return
			}
			assert.Equalf(t, tt.want, got, "createDockerProvider(ctx, 'provider', cfg)")
		})
	}
}

func TestDocker_findContainerIp(t *testing.T) {
	ctx := context.TestContext(nil)

	tests := []struct {
		name            string
		container       *dockerTypes.Container
		recordContainer *ConfigRecordContainer
		want            string
	}{
		{
			name: "SuccessDefault",
			container: &dockerTypes.Container{NetworkSettings: &dockerTypes.SummaryNetworkSettings{
				Networks: map[string]*dockerNetwork.EndpointSettings{"project_other": {IPAddress: "127.0.0.2"}, "project_default": {IPAddress: "127.0.0.1"}},
			}},
			recordContainer: &ConfigRecordContainer{},
			want:            "127.0.0.1",
		},
		{
			name: "SuccessWithSpecificNetwork",
			container: &dockerTypes.Container{NetworkSettings: &dockerTypes.SummaryNetworkSettings{
				Networks: map[string]*dockerNetwork.EndpointSettings{"project_other": {IPAddress: "127.0.0.2"}, "project_default": {IPAddress: "127.0.0.1"}},
			}},
			recordContainer: &ConfigRecordContainer{Network: "project_other"},
			want:            "127.0.0.2",
		},
		{
			name: "SuccessWithUnknownNetwork",
			container: &dockerTypes.Container{NetworkSettings: &dockerTypes.SummaryNetworkSettings{
				Networks: map[string]*dockerNetwork.EndpointSettings{"project_other": {IPAddress: "127.0.0.2"}, "project_default": {IPAddress: "127.0.0.1"}},
			}},
			recordContainer: &ConfigRecordContainer{Network: "unknown"},
			want:            "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Docker{
				id:     "provider",
				logger: ctx.Logger,
			}
			assert.Equalf(t, tt.want, d.findContainerIp(tt.container, tt.recordContainer), "findContainerIp(%v, %v)", tt.container, tt.recordContainer)
		})
	}
}

func TestDocker_formatLabelsToRecords(t *testing.T) {
	ctx := context.TestContext(nil)
	tests := []struct {
		name      string
		container *dockerTypes.Container
		want      []*types.Record
	}{
		{
			name: "Success",
			container: &dockerTypes.Container{
				Names:           []string{"test"},
				NetworkSettings: &dockerTypes.SummaryNetworkSettings{Networks: map[string]*dockerNetwork.EndpointSettings{"project_default": {IPAddress: "127.0.0.1"}, "project_other": {IPAddress: "127.0.0.2"}}},
				Labels: map[string]string{
					fmt.Sprintf("%s.enable", types.AppName):               "true",
					fmt.Sprintf("%s.records.foo.name", types.AppName):     "foo.local",
					fmt.Sprintf("%s.records.foo.type", types.AppName):     "A",
					fmt.Sprintf("%s.records.bar.name", types.AppName):     "bar.local",
					fmt.Sprintf("%s.records.bar.type", types.AppName):     "CNAME",
					fmt.Sprintf("%s.records.bar.value", types.AppName):    "foo.local.",
					fmt.Sprintf("%s.records.foo2.name", types.AppName):    "foo.local",
					fmt.Sprintf("%s.records.foo2.type", types.AppName):    "A",
					fmt.Sprintf("%s.records.foo2.network", types.AppName): "project_other",
				},
			},
			want: []*types.Record{
				{Name: "bar.local", Type: "CNAME", Value: "foo.local."},
				{Name: "foo.local", Type: "A", Value: "127.0.0.1"},
				{Name: "foo.local", Type: "A", Value: "127.0.0.2"},
			},
		},
		{
			name: "FailDecodeLabels",
			container: &dockerTypes.Container{
				Names:           []string{"test"},
				NetworkSettings: &dockerTypes.SummaryNetworkSettings{Networks: map[string]*dockerNetwork.EndpointSettings{"project_default": {IPAddress: "127.0.0.1"}}},
				Labels: map[string]string{
					fmt.Sprintf("%s.enable", types.AppName):            "true",
					fmt.Sprintf("%s.records.foo.name", types.AppName):  "foo.local",
					fmt.Sprintf("%s.records.foo.wrong", types.AppName): "A",
				},
			},
			want: []*types.Record{},
		},
		{
			name: "FailFindIp",
			container: &dockerTypes.Container{
				Names:           []string{"test"},
				NetworkSettings: &dockerTypes.SummaryNetworkSettings{Networks: map[string]*dockerNetwork.EndpointSettings{}},
				Labels: map[string]string{
					fmt.Sprintf("%s.enable", types.AppName):           "true",
					fmt.Sprintf("%s.records.foo.name", types.AppName): "foo.local",
					fmt.Sprintf("%s.records.foo.type", types.AppName): "A",
				},
			},
			want: []*types.Record{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Docker{
				id:     "provider",
				logger: ctx.Logger,
			}
			got := d.formatLabelsToRecords(tt.container)
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func TestDocker_fetchRecords(t *testing.T) {
	ctx := context.TestContext(nil)

	tests := []struct {
		name    string
		mockFn  func(client *mockDocker.MockAPIClient)
		want    types.Records
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Success",
			mockFn: func(client *mockDocker.MockAPIClient) {
				containers := []dockerTypes.Container{
					{
						Names: []string{"test"},
						NetworkSettings: &dockerTypes.SummaryNetworkSettings{
							Networks: map[string]*dockerNetwork.EndpointSettings{"project_default": {IPAddress: "127.0.0.1"}},
						},
						Labels: map[string]string{
							fmt.Sprintf("%s.enable", types.AppName):           "true",
							fmt.Sprintf("%s.records.foo.name", types.AppName): "foo.local",
							fmt.Sprintf("%s.records.foo.type", types.AppName): "A",
						},
					},
				}
				client.EXPECT().ContainerList(gomock.Any(), gomock.Any()).Times(1).Return(containers, nil)
			},
			want:    types.Records{"foo.local._A": {{Name: "foo.local", Type: "A", Value: "127.0.0.1"}}},
			wantErr: assert.NoError,
		},
		{
			name: "Fail",
			mockFn: func(client *mockDocker.MockAPIClient) {
				client.EXPECT().ContainerList(gomock.Any(), gomock.Any()).Times(1).Return(nil, errors.New("fail"))
			},
			want:    types.Records{},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			client := mockDocker.NewMockAPIClient(ctrl)
			tt.mockFn(client)
			d := Docker{
				id:     "provider",
				client: client,
				logger: ctx.Logger,
			}
			got, err := d.fetchRecords()
			if !tt.wantErr(t, err, fmt.Sprintf("fetchRecords()")) {
				return
			}
			assert.Equalf(t, tt.want, got, "fetchRecords()")
		})
	}
}

func TestDocker_listen(t *testing.T) {
	buffer := &bytes.Buffer{}
	ctx := context.TestContext(buffer)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mockDocker.NewMockAPIClient(ctrl)
	chanMsg, chanErr := make(chan dockerEvents.Message), make(chan error)
	client.EXPECT().Events(gomock.Any(), gomock.Any()).Times(1).Return(chanMsg, chanErr)
	d := Docker{
		id:     "provider",
		client: client,
		logger: ctx.Logger,
		done:   ctx.Done(),
	}
	configurationChan := make(chan types.Message, 40)

	go func() {
		assert.NoError(t, d.listen(configurationChan), fmt.Sprintf("listen(chan)"))
	}()

	containers := []dockerTypes.Container{
		{
			Names: []string{"test"},
			NetworkSettings: &dockerTypes.SummaryNetworkSettings{
				Networks: map[string]*dockerNetwork.EndpointSettings{"project_default": {IPAddress: "127.0.0.1"}},
			},
			Labels: map[string]string{
				fmt.Sprintf("%s.enable", types.AppName):           "true",
				fmt.Sprintf("%s.records.foo.name", types.AppName): "foo.local",
				fmt.Sprintf("%s.records.foo.type", types.AppName): "A",
			},
		},
	}
	client.EXPECT().ContainerList(gomock.Any(), gomock.Any()).Times(1).Return(containers, nil)
	chanMsg <- dockerEvents.Message{Type: dockerEvents.ContainerEventType, Action: dockerEvents.ActionStart}
	msg := <-configurationChan
	assert.Equal(t, types.Message{Provider: d, Records: types.Records{"foo.local._A": {{Name: "foo.local", Type: "A", Value: "127.0.0.1"}}}}, msg)

	client.EXPECT().ContainerList(gomock.Any(), gomock.Any()).Times(1).Return(nil, errors.New("fail"))
	chanMsg <- dockerEvents.Message{Type: dockerEvents.ContainerEventType, Action: dockerEvents.ActionStart}
	time.Sleep(100 * time.Millisecond)
	assert.Contains(t, buffer.String(), "error when fetch container records")
	chanErr <- errors.New("fail")
	time.Sleep(100 * time.Millisecond)
	assert.Contains(t, buffer.String(), "error when fetch containers event")
	client.EXPECT().Close().Times(1).Return(nil)
	ctx.Cancel()
	time.Sleep(100 * time.Millisecond)
}

func TestDocker_Provide_Success(t *testing.T) {
	ctx := context.TestContext(nil)
	ctrl := gomock.NewController(t)
	client := mockDocker.NewMockAPIClient(ctrl)
	chanMsg, chanErr := make(chan dockerEvents.Message), make(chan error)
	client.EXPECT().Events(gomock.Any(), gomock.Any()).Times(1).Return(chanMsg, chanErr)
	containers := []dockerTypes.Container{
		{
			Names: []string{"test"},
			NetworkSettings: &dockerTypes.SummaryNetworkSettings{
				Networks: map[string]*dockerNetwork.EndpointSettings{"project_default": {IPAddress: "127.0.0.1"}},
			},
			Labels: map[string]string{
				fmt.Sprintf("%s.enable", types.AppName):           "true",
				fmt.Sprintf("%s.records.foo.name", types.AppName): "foo.local",
				fmt.Sprintf("%s.records.foo.type", types.AppName): "A",
			},
		},
	}
	client.EXPECT().ContainerList(gomock.Any(), gomock.Any()).Times(1).Return(containers, nil)
	d := Docker{
		id:     "test",
		client: client,
		logger: ctx.Logger,
		done:   ctx.Done(),
	}
	configurationChan := make(chan types.Message, 40)
	go func() {
		err := d.Provide(configurationChan)
		assert.NoError(t, err)
	}()
	msg := <-configurationChan
	assert.Equal(t, types.Message{Provider: d, Records: types.Records{"foo.local._A": {{Name: "foo.local", Type: "A", Value: "127.0.0.1"}}}}, msg)
}

func TestDocker_Provide_Fail(t *testing.T) {
	ctx := context.TestContext(nil)
	ctrl := gomock.NewController(t)
	client := mockDocker.NewMockAPIClient(ctrl)

	client.EXPECT().ContainerList(gomock.Any(), gomock.Any()).Times(1).Return(nil, errors.New("fail"))
	d := Docker{
		id:     "test",
		client: client,
		logger: ctx.Logger,
		done:   ctx.Done(),
	}
	configurationChan := make(chan types.Message, 40)
	err := d.Provide(configurationChan)
	assert.Error(t, err)
	assert.Contains(t, "fail", err.Error())
}
