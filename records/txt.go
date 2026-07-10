package records

import (
	"encoding/json"

	"github.com/miekg/dns"
)

func AppendTXTRecords(msg *dns.Msg, qname string, ttl int, value json.RawMessage) (bool, error) {
	var values []string
	if err := json.Unmarshal(value, &values); err != nil {
		return false, err
	}

	if len(values) == 0 {
		return false, nil
	}

	rr := &dns.TXT{
		Hdr: dns.RR_Header{Name: dns.Fqdn(qname), Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: uint32(ttl)},
		Txt: values,
	}
	msg.Answer = append(msg.Answer, rr)

	return true, nil
}
