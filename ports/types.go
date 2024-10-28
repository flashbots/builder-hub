// Package ports contains entry-point related logic for builder-hub. As of now only way to access builder-hub functionality is via http
package ports

import (
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/flashbots/builder-hub/domain"
)

const (
	AttestationTypeHeader string = "X-Flashbots-Attestation-Type"
	MeasurementHeader     string = "X-Flashbots-Measurement"
	ForwardedHeader       string = "X-Forwarded-For"
)

var ErrInvalidAuthData = errors.New("invalid auth data")

type BuilderWithServiceCreds struct {
	IP           string
	Name         string
	ServiceCreds map[string]ServiceCred
}

type ServiceCred struct {
	TLSCert     string `json:"tls_cert,omitempty"`
	ECDSAPubkey string `json:"ecdsa_pubkey,omitempty"`
}

// MarshalJSON is a custom json marshaller. Unfortunately, there seems to be no way to inline map[string]Service when marshalling
// so we need to be careful when adding new fields, since custom json implementation will ignore it by default
func (b BuilderWithServiceCreds) MarshalJSON() ([]byte, error) {
	// Create a map to hold all fields
	m := make(map[string]interface{})

	// Add the IP field
	m["ip"] = b.IP
	m["name"] = b.Name

	// Add all services
	for k, v := range b.ServiceCreds {
		m[k] = v
	}

	// Marshal the map
	return json.Marshal(m)
}

func fromDomainBuilderWithServices(builder domain.BuilderWithServices) BuilderWithServiceCreds {
	b := BuilderWithServiceCreds{}

	b.IP = builder.Builder.IPAddress.String()
	b.Name = builder.Builder.Name
	b.ServiceCreds = make(map[string]ServiceCred)
	for _, v := range builder.Services {
		b.ServiceCreds[v.Service] = ServiceCred{
			TLSCert:     v.TLSCert,
			ECDSAPubkey: hex.EncodeToString(v.ECDSAPubKey),
		}
	}

	return b
}

type Measurement struct {
	Name            string                              `json:"measurement_id"`
	AttestationType string                              `json:"attestation_type"`
	Measurement     map[string]domain.SingleMeasurement `json:"measurement"`
}

func fromDomainMeasurement(measurement domain.Measurement) Measurement {
	m := Measurement{
		Name:            measurement.Name,
		AttestationType: measurement.AttestationType,
		Measurement:     measurement.Measurement,
	}
	return m
}
