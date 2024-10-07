package domain

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
)

var (
	ErrNotFound         = fmt.Errorf("not found")
	ErrIncorrectBuilder = fmt.Errorf("incorrect builder")
)

type Measurement struct {
	Hash            []byte
	AttestationType string
	Measurement     map[string]string
}

// CalculateHash calculates the sha256 hash of the given measurements
// cat measurements.json | jq --sort-keys --compact-output --join-output| sha256sum
func CalculateHash(measurements map[string]string) []byte {
	bts, _ := json.Marshal(measurements)
	resp := sha256.Sum256(bts)
	return resp[:]
}

func NewMeasurement(attestationType string, measurements map[string]string) *Measurement {
	return &Measurement{
		AttestationType: attestationType,
		Measurement:     measurements,
		Hash:            CalculateHash(measurements),
	}
}

type Builder struct {
	Name      string `json:"name"`
	IPAddress net.IP `json:"ip_address"`
	IsActive  bool   `json:"is_active"`
}

type BuilderWithServices struct {
	Builder  Builder
	Services []BuilderServices
}

type BuilderServices struct {
	TLSCert     string
	ECDSAPubKey []byte
	Service     string
}

type BuilderConfig struct {
}
