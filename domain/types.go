// Package domain contains domain area types/functions for builder hub
package domain

import (
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
// Use Expected for a single value, or ExpectedAny for multiple values with OR semantics.
type SingleMeasurement struct {
	Expected    string   `json:"expected,omitempty"`
	ExpectedAny []string `json:"expected_any,omitempty"`
}

// GetExpectedValues returns the list of expected values.
// If ExpectedAny is set, it returns those values. Otherwise, it returns Expected as a single-element slice.
func (s SingleMeasurement) GetExpectedValues() []string {
	if len(s.ExpectedAny) > 0 {
		return s.ExpectedAny
	}
	if s.Expected != "" {
		return []string{s.Expected}
	}
	return nil
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
