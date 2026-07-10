package consulkv

import (
	"context"
	"net"
	"testing"

	"github.com/miekg/dns"
	"github.com/mwantia/coredns-consulkv-plugin/types"
)

func TestGetZoneAndRecordLabelBoundary(t *testing.T) {
	zones := []string{"example.com", "0.168.192.in-addr.arpa"}

	cases := []struct {
		qname    string
		wantZone string
		wantRec  string
	}{
		{"www.example.com.", "example.com", "www"},
		{"example.com.", "example.com", "@"},
		{"a.b.example.com.", "example.com", "a.b"},
		{"5.0.168.192.in-addr.arpa.", "0.168.192.in-addr.arpa", "5"},
		// Look-alike domains sharing the suffix must NOT match.
		{"notexample.com.", "", ""},
		{"badexample.com.", "", ""},
		{"other.org.", "", ""},
	}

	for _, tc := range cases {
		zone, rec := GetZoneAndRecord(zones, tc.qname)
		if zone != tc.wantZone || rec != tc.wantRec {
			t.Errorf("GetZoneAndRecord(%q) = (%q, %q), want (%q, %q)",
				tc.qname, zone, rec, tc.wantZone, tc.wantRec)
		}
	}
}

func TestApplyDefaultsFlattening(t *testing.T) {
	empty := &ConsulKVConfig{}
	empty.ApplyDefaults()
	if empty.Flattening != types.Flattening_Local {
		t.Fatalf("default Flattening = %q, want %q", empty.Flattening, types.Flattening_Local)
	}

	explicit := &ConsulKVConfig{Flattening: types.Flattening_None}
	explicit.ApplyDefaults()
	if explicit.Flattening != types.Flattening_None {
		t.Fatalf("explicit Flattening = %q, want %q (must be preserved)", explicit.Flattening, types.Flattening_None)
	}
}

func TestCNAMEAppliesTo(t *testing.T) {
	cases := []struct {
		qtype uint16
		found bool
		want  bool
	}{
		{dns.TypeCNAME, false, true},
		{dns.TypeA, false, true},
		{dns.TypeAAAA, false, true},
		{dns.TypeTXT, false, true},
		{dns.TypeHTTPS, false, true},
		{dns.TypeHTTPS, true, false},
		{dns.TypeNS, false, false},
		{dns.TypeSRV, false, false},
	}

	for _, tc := range cases {
		if got := cnameAppliesTo(tc.qtype, tc.found); got != tc.want {
			t.Errorf("cnameAppliesTo(%s, %v) = %v, want %v",
				dns.TypeToString[tc.qtype], tc.found, got, tc.want)
		}
	}
}

type answeringHandler struct {
	answers []dns.RR
}

func (h *answeringHandler) Name() string { return "answering" }

func (h *answeringHandler) ServeDNS(_ context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Answer = h.answers
	if err := w.WriteMsg(m); err != nil {
		return dns.RcodeServerFailure, err
	}
	return dns.RcodeSuccess, nil
}

func TestHandleExternalCNAMEDetectsResolution(t *testing.T) {
	answer := &dns.A{
		Hdr: dns.RR_Header{Name: "target.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
		A:   net.ParseIP("192.0.2.1").To4(),
	}
	plug := &ConsulKVPlugin{Next: &answeringHandler{answers: []dns.RR{answer}}}

	msg := new(dns.Msg)
	if !plug.HandleExternalCNAME(context.TODO(), msg, "target.example.", dns.TypeA) {
		t.Fatal("expected external resolution that produced an answer to report success")
	}
}

func TestHandleExternalCNAMENoAnswerReportsFailure(t *testing.T) {
	plug := &ConsulKVPlugin{Next: &answeringHandler{answers: nil}}

	msg := new(dns.Msg)
	if plug.HandleExternalCNAME(context.TODO(), msg, "target.example.", dns.TypeA) {
		t.Fatal("expected external resolution with no answers to report failure")
	}
}
