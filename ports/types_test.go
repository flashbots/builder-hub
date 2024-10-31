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
