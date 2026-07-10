package consulkv

import (
	"time"

	"github.com/hashicorp/consul/api"
)

func (plug *ConsulKVPlugin) Ready() bool {
	if plug.Consul == nil {
		return false
	}

	_, _, err := plug.Consul.Client.Health().Service("consul", "", false, &api.QueryOptions{
		AllowStale:        true,
		UseCache:          true,
		MaxAge:            1 * time.Second,
		RequireConsistent: false,
	})

	return err == nil
}
