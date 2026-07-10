package records

import (
	"encoding/json"
	"net"

	"github.com/miekg/dns"
	"github.com/mwantia/coredns-consulkv-plugin/logging"
)

func AppendARecords(msg *dns.Msg, qname string, ttl int, value json.RawMessage) (bool, error) {
	var ips []string
	if err := json.Unmarshal(value, &ips); err != nil {
		return false, err
	}

	found := false
	for _, ip := range ips {
		parsed := net.ParseIP(ip)
		if parsed == nil || parsed.To4() == nil {
			logging.Log.Warningf("Skipping invalid IPv4 address %q for A record %s", ip, qname)
			continue
		}

		rr := &dns.A{
			Hdr: dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: uint32(ttl)},
			A:   parsed.To4(),
		}
		msg.Answer = append(msg.Answer, rr)
		found = true
	}

	return found, nil
}
