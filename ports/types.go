// Package ports contains entry-point related logic for builder-hub. As of now only way to access builder-hub functionality is via http
package ports

import (
	"encoding/json"
	"errors"
	"net"

	"github.com/ethereum/go-ethereum/common"
	"github.com/flashbots/builder-hub/domain"
)

const (
	AttestationTypeHeader string = "X-Flashbots-Attestation-Type"
	MeasurementHeader     string = "X-Flashbots-Measurement"
	ForwardedHeader       string = "X-Forwarded-For"
)

var (
	ErrInvalidAuthData  = errors.New("invalid auth data")
	ErrInvalidIPAddress = errors.New("invalid ip address")
)

type BuilderWithServiceCreds struct {
	IP           string
	Name         string
	ServiceCreds map[string]ServiceCred
}

type ServiceCred struct {
	TLSCert     string          `json:"tls_cert,omitempty"`
	ECDSAPubkey *common.Address `json:"ecdsa_pubkey_address,omitempty"`
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

func (b *BuilderWithServiceCreds) UnmarshalJSON(data []byte) error {
	// Define a temporary struct to unmarshal known fields
	type Alias struct {
		IP   string `json:"ip"`
		Name string `json:"name"`
	}

	// Unmarshal known fields first
	var alias Alias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	// Copy known fields to the original struct
	b.IP = alias.IP
	b.Name = alias.Name

	// Unmarshal the remaining fields into a map
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return err
	}

	// Remove known fields from the map
	delete(rawMap, "ip")
	delete(rawMap, "name")

	// Initialize the ServiceCreds map
	b.ServiceCreds = make(map[string]ServiceCred)

	// Unmarshal each remaining field into the ServiceCreds map
	for key, rawValue := range rawMap {
		var serviceCred ServiceCred
		if err := json.Unmarshal(rawValue, &serviceCred); err != nil {
			return err
		}
		b.ServiceCreds[key] = serviceCred
	}

	return nil
}

func fromDomainBuilderWithServices(builder domain.BuilderWithServices) BuilderWithServiceCreds {
	b := BuilderWithServiceCreds{}

	b.IP = builder.Builder.IPAddress.String()
	b.Name = builder.Builder.Name
	b.ServiceCreds = make(map[string]ServiceCred)
	for _, v := range builder.Services {
		b.ServiceCreds[v.Service] = ServiceCred{
			TLSCert:     v.TLSCert,
			ECDSAPubkey: v.ECDSAPubKey,
		}
	}

	return b
}

type Measurement struct {
	Name            string                              `json:"measurement_id"`
	AttestationType string                              `json:"attestation_type"`
	Measurements    map[string]domain.SingleMeasurement `json:"measurements"`
}

func fromDomainMeasurement(measurement domain.Measurement) Measurement {
	m := Measurement{
		Name:            measurement.Name,
		AttestationType: measurement.AttestationType,
		Measurements:    measurement.Measurement,
	}
	return m
}

func toDomainMeasurement(measurement Measurement) domain.Measurement {
	m := domain.NewMeasurement(measurement.Name, measurement.AttestationType, measurement.Measurements)
	return *m
}

type Builder struct {
	Name      string `json:"name"`
	IPAddress string `json:"ip_address"`
	Network   string `json:"network"`
}

func toDomainBuilder(builder Builder, enabled bool) (domain.Builder, error) {
	ip := net.ParseIP(builder.IPAddress)
	if ip == nil {
		return domain.Builder{}, ErrInvalidIPAddress
	}

	return domain.Builder{
		Name:      builder.Name,
		IPAddress: ip,
		IsActive:  enabled,
		Network:   builder.Network,
	}, nil
}
