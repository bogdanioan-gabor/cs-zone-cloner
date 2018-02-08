package definition

import (
	"errors"
	"fmt"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

const (
	DefaultScheme  = "http"
	DefaultAddress = "127.0.0.1:8080"
	DefaultPath    = "/client/api"
)

var (
	client *cloudstack.CloudStackClient

	defaultFetchers = []Fetcher{
		fetchPods,
		fetchClusters,
		fetchHosts,
		fetchPrimaryStoragePools,
		fetchSecondaryStoragePools,
		fetchPhysicalNetworks,
		fetchComputeOfferings,
		fetchDiskOfferings,
		fetchGlobalConfigs,
	}
)

type (
	TrafficType struct {
		cloudstack.TrafficType
		Networks map[string]cloudstack.Network
	}
	PhysicalNetwork struct {
		cloudstack.PhysicalNetwork
		TrafficTypes map[string]TrafficType
	}

	ZoneDefinition struct {
		Zone                  cloudstack.Zone
		Pods                  map[string]cloudstack.Pod
		Clusters              map[string]cloudstack.Cluster
		Hosts                 map[string]cloudstack.Host
		PrimaryStoragePools   map[string]cloudstack.StoragePool
		SecondaryStoragePools map[string]cloudstack.ImageStore
		PhysicalNetworks      map[string]PhysicalNetwork
		ComputeOfferings      map[string]cloudstack.ServiceOffering
		DiskOfferings         map[string]cloudstack.DiskOffering
		GlobalConfigs         map[string]cloudstack.Configuration
	}
)

func NewZoneDefinition(zone cloudstack.Zone) *ZoneDefinition {
	zd := &ZoneDefinition{
		Zone:                  zone,
		Pods:                  make(map[string]cloudstack.Pod),
		Clusters:              make(map[string]cloudstack.Cluster),
		Hosts:                 make(map[string]cloudstack.Host),
		PrimaryStoragePools:   make(map[string]cloudstack.StoragePool),
		SecondaryStoragePools: make(map[string]cloudstack.ImageStore),
		PhysicalNetworks:      make(map[string]PhysicalNetwork),
		ComputeOfferings:      make(map[string]cloudstack.ServiceOffering),
		DiskOfferings:         make(map[string]cloudstack.DiskOffering),
		GlobalConfigs:         make(map[string]cloudstack.Configuration),
	}
	return zd
}

type (
	Fetcher func(*ZoneDefinition) error

	Config struct {
		Key      string `json:"key"`
		Secret   string `json:"secret"`
		Scheme   string `json:"scheme"`
		Address  string `json:"address"`
		Path     string `json:"path"`
		ZoneID   string `json:"zoneID"`
		ZoneName string `json:"zoneName"`

		Fetchers []Fetcher `json:"-"`
	}
)

func FetchDefinition(conf Config) (*ZoneDefinition, error) {
	var zone *cloudstack.Zone
	var count int
	var err error

	key := conf.Key
	secret := conf.Secret
	scheme := conf.Scheme
	address := conf.Address
	path := conf.Path
	zoneName := conf.ZoneName
	zoneID := conf.ZoneID
	if key == "" {
		return nil, errors.New("key cannot be empty")
	}
	if secret == "" {
		return nil, errors.New("secret cannot be empty")
	}
	if scheme != "" && conf.Scheme != "http" && conf.Scheme != "https" {
		return nil, errors.New("scheme must be http or https")
	} else if scheme == "" {
		scheme = DefaultScheme
	}
	if address == "" {
		address = DefaultAddress
	}
	if path == "" {
		path = DefaultPath
	}
	if zoneName == "" && zoneID == "" {
		return nil, errors.New("zone name or id must be populated")
	}

	client = cloudstack.NewAsyncClient(fmt.Sprintf("%s://%s%s", scheme, address, path), key, secret, false)
	if zoneID == "" {
		log.Println("Attempting to fetch zone " + zoneName)
		zone, count, err = client.Zone.GetZoneByName(zoneName)
		if err != nil {
			return nil, err
		}
		if count == 0 {
			return nil, fmt.Errorf("zone " + zoneName + " not found")
		}
	} else {
		log.Println("Attempting to fetch zone " + zoneID)
		zone, count, err = client.Zone.GetZoneByID(zoneID)
		if err != nil {
			return nil, err
		}
		if count == 0 {
			return nil, fmt.Errorf("zone " + zoneID + " not found")
		}
	}

	zd := NewZoneDefinition(*zone)

	for _, fetcher := range append(defaultFetchers, conf.Fetchers...) {
		if err = fetcher(zd); err != nil {
			return nil, err
		}
	}

	return zd, nil
}