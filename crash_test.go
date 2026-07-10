package consulkv

import (
	"context"
	"testing"
	"time"

	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"
	"github.com/miekg/dns"
)

type passthroughHandler struct {
	called bool
}

func (h *passthroughHandler) Name() string { return "passthrough" }

func (h *passthroughHandler) ServeDNS(_ context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	h.called = true
	m := new(dns.Msg)
	m.SetReply(r)
	if err := w.WriteMsg(m); err != nil {
		return dns.RcodeServerFailure, err
	}
	return dns.RcodeSuccess, nil
}

// TestServeDNSWithoutConfigDoesNotPanic covers the startup case where the
// Consul '<prefix>/config' key is absent: the plugin must serve requests
// (passing them to the next plugin) instead of dereferencing a nil config.
func TestServeDNSWithoutConfigDoesNotPanic(t *testing.T) {
	next := &passthroughHandler{}
	plug := &ConsulKVPlugin{Next: next}
	// Deliberately no SetConfig: GetConfig must return a usable empty config.

	req := new(dns.Msg)
	req.SetQuestion("example.org.", dns.TypeA)
	rec := dnstest.NewRecorder(&test.ResponseWriter{})

	code, err := plug.ServeDNS(context.TODO(), rec, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != dns.RcodeSuccess {
		t.Fatalf("code = %d, want %d (passthrough)", code, dns.RcodeSuccess)
	}
	if !next.called {
		t.Fatal("expected the query to be passed to the next plugin")
	}
}

// TestCreateQueryOptionsMaxAgeSeconds pins the documented unit for max_age:
// the configured integer is seconds, not nanoseconds.
func TestCreateQueryOptionsMaxAgeSeconds(t *testing.T) {
	age := 60
	opts := CreateQueryOptions(&ConsulKVCache{MaxAge: &age})

	if opts.MaxAge != 60*time.Second {
		t.Fatalf("MaxAge = %v, want %v", opts.MaxAge, 60*time.Second)
	}
}

// TestRegisterMetricsIsIdempotent guards against the duplicate-registration
// panic CoreDNS would otherwise hit when setup runs again on a reload.
func TestRegisterMetricsIsIdempotent(t *testing.T) {
	registerMetrics()
	registerMetrics()
}

// TestConfigSwapIsRaceFree exercises concurrent reads and writes of the config
// so `go test -race` flags any regression to unsynchronized access.
func TestConfigSwapIsRaceFree(t *testing.T) {
	plug := &ConsulKVPlugin{}
	plug.SetConfig(&ConsulKVConfig{Zones: []string{"a.example."}})

	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			plug.SetConfig(&ConsulKVConfig{Zones: []string{"b.example."}})
		}
		close(done)
	}()

	for i := 0; i < 1000; i++ {
		_ = plug.GetConfig().Zones
	}
	<-done
}
