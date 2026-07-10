package records

import "encoding/json"

type Record struct {
	TTL     *int `json:"ttl"`
	Records []struct {
		Type  string          `json:"type"`
		Value json.RawMessage `json:"value"`
	} `json:"records"`
}

func GetRecordTTL(record *Record) int {
	if record.TTL != nil {
		return *record.TTL
	}

	return 3600 // Default TTL
}
