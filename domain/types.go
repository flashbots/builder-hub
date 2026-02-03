// Package domain contains domain area types/functions for builder hub
package domain

import (
	"encoding/json"
	"errors"
	"net"

	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrIncorrectBuilder   = errors.New("incorrect builder")
	ErrInvalidMeasurement = errors.New("no such active measurement found")
)

const ProductionNetwork = "production"

const EventGetConfig = "GetConfig"

type Measurement struct {
	Name            string
	AttestationType string
	Measurement     map[string]SingleMeasurement
}

// SingleMeasurement represents a single measurement with one or more expected values.
// When multiple values are provided, any matching value is accepted (OR semantics).
type SingleMeasurement struct {
	Expected []string
}

// UnmarshalJSON implements custom unmarshaling to support both:
// - {"expected": "value"} (backwards compatible single string)
// - {"expected": ["value1", "value2"]} (new list format)
func (s *SingleMeasurement) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as object with expected field
	var raw struct {
		Expected json.RawMessage `json:"expected"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Try to unmarshal expected as a string first (backwards compatibility)
	var singleValue string
	if err := json.Unmarshal(raw.Expected, &singleValue); err == nil {
		s.Expected = []string{singleValue}
		return nil
	}

	// Try to unmarshal expected as an array of strings
	var multiValue []string
	if err := json.Unmarshal(raw.Expected, &multiValue); err != nil {
		return err
	}
	s.Expected = multiValue
	return nil
}

// MarshalJSON implements custom marshaling to output:
// - {"expected": "value"} when there's exactly one value (backwards compatible)
// - {"expected": ["value1", "value2"]} when there are multiple values
func (s SingleMeasurement) MarshalJSON() ([]byte, error) {
	if len(s.Expected) == 1 {
		return json.Marshal(struct {
			Expected string `json:"expected"`
		}{Expected: s.Expected[0]})
	}
	return json.Marshal(struct {
		Expected []string `json:"expected"`
	}{Expected: s.Expected})
}

func NewMeasurement(name, attestationType string, measurements map[string]SingleMeasurement) *Measurement {
	return &Measurement{
		AttestationType: attestationType,
		Measurement:     measurements,
		Name:            name,
	}
}

type Builder struct {
	Name      string `json:"name"`
	IPAddress net.IP `json:"ip_address"`
	IsActive  bool   `json:"is_active"`
	Network   string `json:"network"`
	DNSName   string `json:"dns_name"`
}

type BuilderWithServices struct {
	Builder  Builder
	Services []BuilderServices
}

type BuilderServices struct {
	TLSCert     string
	ECDSAPubKey *common.Address
	Service     string
}

func Bytes2Address(b []byte) *common.Address {
	if len(b) == 0 {
		return nil
	}
	addr := common.BytesToAddress(b)
	return &addr
}
