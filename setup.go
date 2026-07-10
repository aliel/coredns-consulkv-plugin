package consulkv

import (
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/mwantia/coredns-consulkv-plugin/logging"
)

var soaSerial = uint32(time.Now().Unix())

func init() {
	plugin.Register("consulkv", setup)
}

func setup(c *caddy.Controller) error {
	c.OnStartup(func() error {
		registerMetrics()
		return nil
	})

	conf, err := CreatePlugin(c)
	if err != nil {
		return plugin.Error("consulkv", err)
	}

	if !conf.Consul.DisableWatch {
		plan, err := conf.Consul.WatchConsulConfig(conf.SetConfig)
		if err != nil {
			logging.Log.Warningf("Unable to create Consul watcher for '%s/config'", conf.Consul.KVPrefix)
		} else {
			c.OnShutdown(func() error {
				plan.Stop()
				return nil
			})
		}
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		conf.Next = next
		return conf
	})

	return nil
}
