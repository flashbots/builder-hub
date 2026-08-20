// Package domain contains domain area types/functions for builder hub
package domain

import (
	"encoding/hex"
	"errors"
	"fmt"
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
	DcapImageHashes *DcapImageHashes
}

// DcapImageHashes contains the image-specific SHA-384 hashes used by portable
// DCAP measurement policies.
type DcapImageHashes struct {
	UKIAuthenticode    string `json:"uki_authenticode"`
	KernelAuthenticode string `json:"kernel_authenticode"`
	CmdlineHash        string `json:"cmdline_hash"`
	InitrdHash         string `json:"initrd_hash"`
	GPTDiskGUIDHash    string `json:"gpt_disk_guid_hash"`
}

type ExpectedMeasurementType string

const (
	ExpectedMeasurementLegacy        ExpectedMeasurementType = "legacy"
	ExpectedMeasurementDCAP          ExpectedMeasurementType = "dcap"
	ExpectedMeasurementAzure         ExpectedMeasurementType = "azure"
	ExpectedMeasurementImage         ExpectedMeasurementType = "image"
	ExpectedMeasurementNoAttestation ExpectedMeasurementType = "no_attestation"
)

// SuppliedMeasurements represents either a legacy set of observed register
// values or the expected policy selected by a current attestation proxy.
type SuppliedMeasurements struct {
	Type            ExpectedMeasurementType
	Registers       map[string][]string
	DcapImageHashes *DcapImageHashes
	Legacy          map[string]string
}

// Validate checks that every portable image measurement is a SHA-384 value.
func (d DcapImageHashes) Validate() error {
	values := []struct {
		name  string
		value string
	}{
		{name: "uki_authenticode", value: d.UKIAuthenticode},
		{name: "kernel_authenticode", value: d.KernelAuthenticode},
		{name: "cmdline_hash", value: d.CmdlineHash},
		{name: "initrd_hash", value: d.InitrdHash},
		{name: "gpt_disk_guid_hash", value: d.GPTDiskGUIDHash},
	}

	for _, item := range values {
		decoded, err := hex.DecodeString(item.value)
		if err != nil || len(decoded) != 48 {
			return fmt.Errorf("%s must be a 48-byte hexadecimal value", item.name)
		}
	}
	return nil
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
		DcapImageHashes: nil,
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
	Region      string
}

func Bytes2Address(b []byte) *common.Address {
	if len(b) == 0 {
		return nil
	}
	addr := common.BytesToAddress(b)
	return &addr
}
