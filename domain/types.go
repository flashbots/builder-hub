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

type Measurement struct {
	Name            string
	AttestationType string
	Measurement     map[string]SingleMeasurement
}

type SingleMeasurement struct {
	Expected string `json:"expected"`
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
