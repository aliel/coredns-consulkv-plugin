package consulkv

import (
	"sync/atomic"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin"
	"github.com/mwantia/coredns-consulkv-plugin/logging"
	"github.com/mwantia/coredns-consulkv-plugin/types"
)

type ConsulKVPlugin struct {
	Next   plugin.Handler
	Consul *ConsulConfig
	config atomic.Pointer[ConsulKVConfig]
}

// GetConfig returns the current configuration snapshot. It never returns nil,
// so callers can read it without a nil check even before the first config load.
func (plug *ConsulKVPlugin) GetConfig() *ConsulKVConfig {
	if config := plug.config.Load(); config != nil {
		return config
	}

	return &ConsulKVConfig{}
}

// SetConfig atomically replaces the active configuration. It is safe to call
// from the Consul watch goroutine while requests read the config concurrently.
func (plug *ConsulKVPlugin) SetConfig(config *ConsulKVConfig) {
	plug.config.Store(config)
}

// ApplyDefaults fills in values the operator may omit. Flattening defaults to
// "local" (the documented default); its zero value "" would otherwise fall
// through the disable-flattening guard and behave like "local" only by accident.
func (config *ConsulKVConfig) ApplyDefaults() {
	if config.Flattening == "" {
		config.Flattening = types.Flattening_Local
	}
}

type ConsulKVConfig struct {
	ZonePrefix  string               `json:"zone_prefix"`
	Zones       []string             `json:"zones"`
	Flattening  types.FlatteningType `json:"flattening,omitempty"`
	NoCache     bool                 `json:"no_cache,omitempty"`
	ConsulCache *ConsulKVCache       `json:"consul_cache,omitempty"`
}

type ConsulKVCache struct {
	UseCache   *bool `json:"use_cache,omitempty"`
	MaxAge     *int  `json:"max_age"`
	Consistent *bool `json:"consistent"`
	AllowStale *bool `json:"allowstale"`
}

func CreatePlugin(c *caddy.Controller) (*ConsulKVPlugin, error) {
	plug := &ConsulKVPlugin{}

	consul, err := CreateConsulConfig(c)
	if err != nil {
		return nil, err
	}

	config, err := consul.GetConfigFromConsul()
	if err != nil {
		return nil, err
	}

	if config == nil {
		logging.Log.Warningf("No configuration found at '%s/config'; starting with an empty zone set", consul.KVPrefix)
		config = &ConsulKVConfig{}
	}

	plug.Consul = consul
	plug.SetConfig(config)

	return plug, nil
}
