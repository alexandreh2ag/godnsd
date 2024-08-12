package provider

import (
	stdContext "context"
	"fmt"
	"github.com/alexandreh2ag/go-dns-discover/config"
	"github.com/alexandreh2ag/go-dns-discover/context"
	"github.com/alexandreh2ag/go-dns-discover/types"
	dockerTypes "github.com/docker/docker/api/types"
	dockerContainer "github.com/docker/docker/api/types/container"
	dockerEvents "github.com/docker/docker/api/types/events"
	dockerTypesFilters "github.com/docker/docker/api/types/filters"
	docketClient "github.com/docker/docker/client"
	"github.com/traefik/paerser/parser"
	"log/slog"
	"regexp"
	"slices"
)

func init() {
	FactoryProviderMapping[dockerKeyType] = createDockerProvider
}

const (
	dockerKeyType             = "docker"
	defaultNetworkName        = "bridge"
	defaultComposeNetworkName = "default"
)

var (
	_              types.Provider = &Docker{}
	dockerClientFn                = func() (docketClient.APIClient, error) {
		return docketClient.NewClientWithOpts(docketClient.FromEnv, docketClient.WithAPIVersionNegotiation())
	}
)

type ConfigRecordContainer struct {
	Name    string
	Type    string
	Value   string
	Network string
}

type ConfigContainer struct {
	Enable  bool
	Records map[string]*ConfigRecordContainer
}

type Docker struct {
	id     string
	client docketClient.APIClient
	logger *slog.Logger
	done   chan bool
}

func (d Docker) GetId() string {
	return d.id
}

func (d Docker) GetType() string {
	return dockerKeyType
}

func (d Docker) Provide(configurationChan chan<- types.Message) error {
	records, err := d.fetchRecords()
	if err != nil {
		return err
	}
	configurationChan <- types.Message{Provider: d, Records: records}
	return d.listen(configurationChan)
}

func (d Docker) listen(configurationChan chan<- types.Message) error {
	events, errs := d.client.Events(stdContext.Background(), dockerEvents.ListOptions{})
	for {
		select {
		case errEvent := <-errs:
			d.logger.Error(fmt.Sprintf("error when fetch containers event: %s", errEvent.Error()), "provider-type", d.GetType(), "provider-id", d.GetId())
		case msg := <-events:
			if msg.Type == dockerEvents.ContainerEventType && slices.Contains([]dockerEvents.Action{dockerEvents.ActionDie, dockerEvents.ActionStart, dockerEvents.ActionKill, dockerEvents.ActionRestart, dockerEvents.ActionStop}, msg.Action) {
				d.logger.Debug(fmt.Sprintf("event recived"), "provider-type", d.GetType(), "provider-id", d.GetId())
				records, err := d.fetchRecords()
				if err != nil {
					d.logger.Error(fmt.Sprintf("error when fetch container records: %s", err.Error()))
					continue
				}
				configurationChan <- types.Message{Provider: d, Records: records}
			}
		case <-d.done:
			return d.client.Close()
		}
	}
}

func (d Docker) fetchRecords() (types.Records, error) {
	records := types.Records{}
	listOpt := dockerContainer.ListOptions{Filters: dockerTypesFilters.NewArgs()}
	listOpt.Filters.Add("label", fmt.Sprintf("%s.enable=true", types.AppName))
	containers, err := d.client.ContainerList(stdContext.Background(), listOpt)
	if err != nil {
		return records, err
	}

	for _, container := range containers {
		recordsContainer := d.formatLabelsToRecords(&container)
		for _, record := range recordsContainer {
			key := types.FormatRecordKey(record.Name, record.Type)
			if _, ok := records[key]; !ok {
				records[key] = []*types.Record{}
			}
			records[key] = append(records[key], record)
		}
	}

	return records, nil
}

func (d Docker) formatLabelsToRecords(container *dockerTypes.Container) []*types.Record {
	records := []*types.Record{}

	recordsContainer := &ConfigContainer{}
	err := parser.Decode(container.Labels, recordsContainer, types.AppName, types.AppName)
	if err != nil {
		d.logger.Error(fmt.Sprintf("failed to decode labels for docker container %s", container.Names[0]))
		return records
	}

	for key, recordContainer := range recordsContainer.Records {
		record := &types.Record{Name: recordContainer.Name, Type: recordContainer.Type, Value: recordContainer.Value}
		if recordContainer.Value == "" && recordContainer.Type == "A" {
			record.Value = d.findContainerIp(container, recordContainer)
			if record.Value == "" {
				d.logger.Error(fmt.Sprintf("failed to find container ip for container %s, label %s", container.Names[0], key))
				continue
			}
		}
		records = append(records, record)
	}

	return records
}

func (d Docker) findContainerIp(container *dockerTypes.Container, recordContainer *ConfigRecordContainer) string {
	dockerComposeProjectName := container.Labels["com.docker.compose.project"]
	regexNetwork := regexp.MustCompile(fmt.Sprintf("^(%s|%s_%s)$", defaultNetworkName, dockerComposeProjectName, defaultComposeNetworkName))
	if recordContainer.Network != "" && recordContainer.Network != defaultNetworkName {
		regexNetwork = regexp.MustCompile(fmt.Sprintf("^(%s|%s_%s)$", recordContainer.Network, dockerComposeProjectName, recordContainer.Network))
	}
	for networkName, network := range container.NetworkSettings.Networks {
		if regexNetwork.MatchString(networkName) {
			return network.IPAddress
		}
	}
	return ""
}

func createDockerProvider(ctx *context.Context, id string, cfg config.Provider) (types.Provider, error) {
	client, err := dockerClientFn()
	if err != nil {
		return nil, err
	}
	instance := &Docker{
		id:     id,
		logger: ctx.Logger,
		client: client,
		done:   ctx.Done(),
	}
	return instance, nil
}
