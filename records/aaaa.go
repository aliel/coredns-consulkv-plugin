package records

import (
	"encoding/json"
	"net"

	"github.com/miekg/dns"
	"github.com/mwantia/coredns-consulkv-plugin/logging"
)

func AppendAAAARecords(msg *dns.Msg, qname string, ttl int, value json.RawMessage) (bool, error) {
	var ips []string
	if err := json.Unmarshal(value, &ips); err != nil {
		return false, err
	}

	found := false
	for _, ip := range ips {
		parsed := net.ParseIP(ip)
		if parsed == nil || parsed.To4() != nil || parsed.To16() == nil {
			logging.Log.Warningf("Skipping invalid IPv6 address %q for AAAA record %s", ip, qname)
			continue
		}

		rr := &dns.AAAA{
			Hdr:  dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: uint32(ttl)},
			AAAA: parsed.To16(),
		}
		msg.Answer = append(msg.Answer, rr)
		found = true
	}

	return found, nil
}
