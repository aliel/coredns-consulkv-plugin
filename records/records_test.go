package records

import (
	"encoding/json"
	"testing"

	"github.com/miekg/dns"
)

func TestAppendARecordsSkipsInvalid(t *testing.T) {
	msg := new(dns.Msg)
	// Mix of a valid IPv4, garbage, and an IPv6 (not valid for an A record).
	ok, err := AppendARecords(msg, "host.example.", 60, json.RawMessage(`["192.0.2.1","not-an-ip","2001:db8::1"]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected a valid A record to be appended")
	}
	if len(msg.Answer) != 1 {
		t.Fatalf("Answer count = %d, want 1 (only the valid IPv4)", len(msg.Answer))
	}
	if a, okType := msg.Answer[0].(*dns.A); !okType || a.A.String() != "192.0.2.1" {
		t.Fatalf("Answer[0] = %v, want A 192.0.2.1", msg.Answer[0])
	}
}

func TestAppendARecordsAllInvalidReportsNotFound(t *testing.T) {
	msg := new(dns.Msg)
	ok, err := AppendARecords(msg, "host.example.", 60, json.RawMessage(`["nope","2001:db8::1"]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected found=false when no valid IPv4 is present")
	}
	if len(msg.Answer) != 0 {
		t.Fatalf("Answer count = %d, want 0", len(msg.Answer))
	}
}

func TestAppendAAAARecordsRejectsIPv4(t *testing.T) {
	msg := new(dns.Msg)
	ok, err := AppendAAAARecords(msg, "host.example.", 60, json.RawMessage(`["2001:db8::1","192.0.2.1"]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected a valid AAAA record to be appended")
	}
	if len(msg.Answer) != 1 {
		t.Fatalf("Answer count = %d, want 1 (IPv4 must be rejected)", len(msg.Answer))
	}
}

func TestAppendTXTRecords(t *testing.T) {
	msg := new(dns.Msg)
	ok, err := AppendTXTRecords(msg, "host.example.", 60, json.RawMessage(`["hello","world"]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok || len(msg.Answer) != 1 {
		t.Fatalf("ok=%v Answer=%d, want true and 1", ok, len(msg.Answer))
	}
	txt, okType := msg.Answer[0].(*dns.TXT)
	if !okType || len(txt.Txt) != 2 {
		t.Fatalf("Answer[0] = %v, want TXT with 2 strings", msg.Answer[0])
	}
}

func TestAppendTXTRecordsEmpty(t *testing.T) {
	msg := new(dns.Msg)
	ok, err := AppendTXTRecords(msg, "host.example.", 60, json.RawMessage(`[]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok || len(msg.Answer) != 0 {
		t.Fatalf("ok=%v Answer=%d, want false and 0", ok, len(msg.Answer))
	}
}
