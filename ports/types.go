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

var (
	ErrInvalidAuthData = errors.New("invalid auth data")
)

type BuilderWithServiceCreds struct {
	Ip           string
	ServiceCreds map[string]ServiceCred
}

type ServiceCred struct {
	TlsCert     string `json:"tls_cert,omitempty"`
	EcdsaPubkey string `json:"ecdsa_pubkey,omitempty"`
}

// MarshalJSON is a custom json marshaller. Unfortunately, there seems to be no way to inline map[string]Service when marshalling
// so we need to be careful when adding new fields, since custom json implementation will ignore it by default
func (b BuilderWithServiceCreds) MarshalJSON() ([]byte, error) {
	// Create a map to hold all fields
	m := make(map[string]interface{})

	// Add the Ip field
	m["ip"] = b.Ip

	// Add all services
	for k, v := range b.ServiceCreds {
		m[k] = v
	}

	// Marshal the map
	return json.Marshal(m)
}

func fromDomainBuilderWithServices(builder *domain.BuilderWithServices) BuilderWithServiceCreds {
	b := BuilderWithServiceCreds{}

	b.Ip = builder.Builder.IPAddress.String()
	b.ServiceCreds = make(map[string]ServiceCred)
	for _, v := range builder.Services {
		b.ServiceCreds[v.Service] = ServiceCred{
			TlsCert:     v.TLSCert,
			EcdsaPubkey: hex.EncodeToString(v.ECDSAPubKey),
		}
	}

	return b
}

type Measurement struct {
	Hash            string            `json:"measurement_hash"`
	AttestationType string            `json:"attestation_type"`
	Measurement     map[string]string `json:"measurement"`
}

func fromDomainMeasurement(measurement *domain.Measurement) Measurement {
	m := Measurement{
		Hash:            hex.EncodeToString(measurement.Hash),
		AttestationType: measurement.AttestationType,
		Measurement:     measurement.Measurement,
	}
	return m
}
