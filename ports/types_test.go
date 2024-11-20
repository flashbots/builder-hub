package ports

import (
	"encoding/json"
	"testing"
)

func TestServiceCreds(t *testing.T) {
	str := `{"tls_cert":"something", "ecdsa_pubkey_address":"0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"}`
	var sc ServiceCred
	err := json.Unmarshal([]byte(str), &sc)
	if err != nil {
		t.Error("Failed to unmarshal ServiceCred", err)
	}
	if sc.TLSCert != "something" {
		t.Error("Failed to unmarshal TLS cert")
	}
	if sc.ECDSAPubkey.Hex() != "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2" {
		t.Error("Failed to unmarshal ECDSA pubkey")
	}
}

func TestServiceCredsNil(t *testing.T) {
	str := `{"tls_cert":"something"}`
	var sc ServiceCred
	err := json.Unmarshal([]byte(str), &sc)
	if err != nil {
		t.Error("Failed to unmarshal ServiceCred", err)
	}
	if sc.TLSCert != "something" {
		t.Error("Failed to unmarshal TLS cert")
	}
	if sc.ECDSAPubkey != nil {
		t.Error("Failed to unmarshal ECDSA pubkey")
	}
}

func TestUnmarshalBuilders(t *testing.T) {
	val := []byte(`[{"ip":"127.0.0.1","name":"test_builder_1","rbuilder":{"tls_cert":"test-cert-no-validation","ecdsa_pubkey_address":"0x1234567890123456789012345678901234567890"}}]`)
	var builders []BuilderWithServiceCreds
	err := json.Unmarshal(val, &builders)
	if err != nil {
		t.Error("Failed to unmarshal builders", err)
	}
	if len(builders) != 1 {
		t.Error("Failed to unmarshal builders")
	}
	if builders[0].ServiceCreds["rbuilder"].TLSCert != "test-cert-no-validation" {
		t.Error("Failed to unmarshal TLS cert")
	}
}
