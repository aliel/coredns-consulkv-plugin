package consulkv

import (
	"context"
	"encoding/json"

	"github.com/miekg/dns"
	"github.com/mwantia/coredns-consulkv-plugin/logging"
	"github.com/mwantia/coredns-consulkv-plugin/records"
)

// appendRequest carries everything an appender needs to add records of a single
// type to the response being built.
type appendRequest struct {
	msg   *dns.Msg
	qname string
	qtype uint16
	ttl   int
	value json.RawMessage
	soa   *records.SOARecord
	found bool
}

// recordAppenders maps a stored record type to the logic that appends it. Each
// appender first checks whether it applies to the query type and returns
// (false, nil) when it does not, so HandleRecord stays a simple dispatch loop.
//
// CNAME is handled separately in HandleRecord: it recurses back into
// HandleRecord for flattening, which would create a static initialization cycle
// if referenced from this package-level map.
var recordAppenders = map[string]func(*appendRequest) (bool, error){
	"NS": func(r *appendRequest) (bool, error) {
		if r.qtype == dns.TypeNS {
			return records.AppendNSRecords(r.msg, r.qname, r.ttl, r.value)
		}
		return false, nil
	},
	"SVCB": func(r *appendRequest) (bool, error) {
		if r.qtype == dns.TypeSVCB {
			return records.AppendSVCBRecords(r.msg, r.qname, r.ttl, r.value, dns.TypeSVCB)
		}
		return false, nil
	},
	"HTTPS": func(r *appendRequest) (bool, error) {
		if r.qtype == dns.TypeHTTPS {
			return records.AppendSVCBRecords(r.msg, r.qname, r.ttl, r.value, dns.TypeHTTPS)
		}
		return false, nil
	},
	"SOA": func(r *appendRequest) (bool, error) {
		if r.qtype == dns.TypeSOA || r.qtype == dns.TypeANY {
			return records.AppendSOARecord(r.msg, r.qname, r.soa), nil
		}
		return false, nil
	},
	"A": func(r *appendRequest) (bool, error) {
		if r.qtype == dns.TypeA || (r.qtype == dns.TypeHTTPS && !r.found) {
			return records.AppendARecords(r.msg, r.qname, r.ttl, r.value)
		}
		return false, nil
	},
	"AAAA": func(r *appendRequest) (bool, error) {
		if r.qtype == dns.TypeAAAA || (r.qtype == dns.TypeHTTPS && !r.found) {
			return records.AppendAAAARecords(r.msg, r.qname, r.ttl, r.value)
		}
		return false, nil
	},
	"PTR": func(r *appendRequest) (bool, error) {
		if r.qtype != dns.TypePTR {
			return false, nil
		}
		if records.IsDnsSdQuery(r.qname) {
			return records.AppendDnsSdPTRRecords(r.msg, r.qname, r.ttl, r.value)
		}
		return records.AppendPTRRecords(r.msg, r.qname, r.ttl, r.value)
	},
	"SRV": func(r *appendRequest) (bool, error) {
		if r.qtype == dns.TypeSRV {
			return records.AppendSRVRecords(r.msg, r.qname, r.ttl, r.value)
		}
		return false, nil
	},
	"TXT": func(r *appendRequest) (bool, error) {
		if r.qtype == dns.TypeTXT {
			return records.AppendTXTRecords(r.msg, r.qname, r.ttl, r.value)
		}
		return false, nil
	},
}

func (plug *ConsulKVPlugin) HandleRecord(ctx context.Context, msg *dns.Msg, qname string, qtype uint16, record *records.Record) bool {
	ttl := GetDefaultTTL(record)

	logging.Log.Debugf("Amount of available records: %v", len(record.Records))

	config := plug.GetConfig()
	zname, _ := GetZoneAndRecord(config.Zones, qname)
	soa, err := plug.Consul.GetSOARecordFromConsul(zname, config.ConsulCache)
	if err != nil {
		logging.Log.Errorf("Error loading SOA record: %v", err)
		IncrementMetricsPluginErrorsTotal("SOA_GET")
	}

	found := false
	for _, rec := range record.Records {
		logging.Log.Debugf("Searching record for type %s", rec.Type)

		if rec.Type == "CNAME" {
			if cnameAppliesTo(qtype, found) &&
				plug.AppendCNAMERecords(ctx, msg, qname, qtype, ttl, rec.Value) {
				found = true
			}
			continue
		}

		appender, ok := recordAppenders[rec.Type]
		if !ok {
			continue
		}

		req := &appendRequest{msg, qname, qtype, ttl, rec.Value, soa, found}
		matched, err := appender(req)
		if err != nil {
			logging.Log.Errorf("Error parsing JSON for %s record: %v", rec.Type, err)
			IncrementMetricsPluginErrorsTotal("JSON_UNMARSHAL")
			continue
		}
		if matched {
			found = true
		}
	}

	return plug.finalizeAnswer(msg, qname, qtype, soa, found)
}

// cnameAppliesTo reports whether a CNAME record should be processed for the
// given query type. A/AAAA/TXT queries flatten through the CNAME, and HTTPS
// queries follow it only when no direct match has been found yet.
func cnameAppliesTo(qtype uint16, found bool) bool {
	return qtype == dns.TypeCNAME || qtype == dns.TypeA || qtype == dns.TypeAAAA ||
		qtype == dns.TypeTXT || (qtype == dns.TypeHTTPS && !found)
}

// finalizeAnswer applies the cross-record fallbacks: treat any answer as a match
// for service-binding queries, and add an authority SOA when nothing matched.
func (plug *ConsulKVPlugin) finalizeAnswer(msg *dns.Msg, qname string, qtype uint16, soa *records.SOARecord, found bool) bool {
	if (qtype == dns.TypeSVCB || qtype == dns.TypeHTTPS) && !found && len(msg.Answer) > 0 {
		found = true
	}

	if !found && soa != nil && qtype != dns.TypeSOA && qtype != dns.TypeANY {
		records.AppendSOAToAuthority(msg, qname, soa)
	}

	return found
}
