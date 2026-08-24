package ports

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/flashbots/builder-hub/domain"
	"github.com/stretchr/testify/require"
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

func TestUnmarshalBuildersWithRegion(t *testing.T) {
	val := []byte(`[{"ip":"127.0.0.1","name":"test_builder_1","rbuilder":{"tls_cert":"test-cert","ecdsa_pubkey_address":"0x1234567890123456789012345678901234567890","region":"us-east-1"}}]`)
	var builders []BuilderWithServiceCreds
	err := json.Unmarshal(val, &builders)
	if err != nil {
		t.Fatalf("Failed to unmarshal builders: %v", err)
	}
	if len(builders) != 1 {
		t.Fatal("Expected 1 builder")
	}
	if builders[0].ServiceCreds["rbuilder"].Region != "us-east-1" {
		t.Errorf("Expected region us-east-1, got %s", builders[0].ServiceCreds["rbuilder"].Region)
	}
	if builders[0].ServiceCreds["rbuilder"].TLSCert != "test-cert" {
		t.Error("Failed to unmarshal TLS cert")
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

func TestParseMeasurementHeader(t *testing.T) {
	t.Run("legacy", func(t *testing.T) {
		measurement, err := parseMeasurementHeader(`{"4":"aaaa"}`, "custom-attestation")
		require.NoError(t, err)
		require.Equal(t, domain.ExpectedMeasurementLegacy, measurement.Type)
		require.Equal(t, map[string]string{"4": "aaaa"}, measurement.Legacy)
	})

	t.Run("DCAP", func(t *testing.T) {
		measurement, err := parseMeasurementHeader(`{"type":"dcap","measurements":{"0":["aaaa"],"3":["bbbb","cccc"]}}`, "dcap-tdx")
		require.NoError(t, err)
		require.Equal(t, domain.ExpectedMeasurementDCAP, measurement.Type)
		require.Equal(t, []string{"bbbb", "cccc"}, measurement.Registers["3"])
	})

	t.Run("Azure", func(t *testing.T) {
		measurement, err := parseMeasurementHeader(`{"type":"azure","measurements":{}}`, "azure-tdx")
		require.NoError(t, err)
		require.NotNil(t, measurement.Registers)
		require.Empty(t, measurement.Registers)
	})

	for _, attestationType := range []string{"dcap-tdx", "gcp-tdx"} {
		t.Run("image "+attestationType, func(t *testing.T) {
			hashes := validImageHashes()
			header, err := json.Marshal(map[string]any{
				"type":         "image",
				"measurements": hashes,
			})
			require.NoError(t, err)

			measurement, err := parseMeasurementHeader(string(header), attestationType)
			require.NoError(t, err)
			require.Equal(t, &hashes, measurement.DcapImageHashes)
		})
	}

	t.Run("no attestation", func(t *testing.T) {
		measurement, err := parseMeasurementHeader(`{"type":"no_attestation"}`, "none")
		require.NoError(t, err)
		require.Equal(t, domain.ExpectedMeasurementNoAttestation, measurement.Type)
	})

	t.Run("typed header does not fall back", func(t *testing.T) {
		_, err := parseMeasurementHeader(`{"type":"future","4":"aaaa"}`, "azure-tdx")
		require.Error(t, err)
	})

	t.Run("rejects null", func(t *testing.T) {
		_, err := parseMeasurementHeader(`null`, "dcap-tdx")
		require.Error(t, err)
	})

	t.Run("rejects mismatched attestation type", func(t *testing.T) {
		_, err := parseMeasurementHeader(`{"type":"image","measurements":{}}`, "azure-tdx")
		require.Error(t, err)
	})

	t.Run("rejects empty register alternatives", func(t *testing.T) {
		_, err := parseMeasurementHeader(`{"type":"dcap","measurements":{"0":[]}}`, "dcap-tdx")
		require.Error(t, err)
	})

	t.Run("rejects measurements for no attestation", func(t *testing.T) {
		_, err := parseMeasurementHeader(`{"type":"no_attestation","measurements":{}}`, "none")
		require.Error(t, err)
	})
}

func TestToDomainMeasurementPortablePolicy(t *testing.T) {
	hashes := validImageHashes()
	measurement, err := toDomainMeasurement(Measurement{
		Name:            "portable",
		AttestationType: "dcap-tdx",
		DcapImageHashes: &hashes,
	})
	require.NoError(t, err)
	require.Nil(t, measurement.Measurement)
	require.Equal(t, &hashes, measurement.DcapImageHashes)

	_, err = toDomainMeasurement(Measurement{
		Name:            "invalid",
		AttestationType: "azure-tdx",
		DcapImageHashes: &hashes,
	})
	require.Error(t, err)

	_, err = toDomainMeasurement(Measurement{
		Name:            "both",
		AttestationType: "gcp-tdx",
		Measurements:    map[string]domain.SingleMeasurement{},
		DcapImageHashes: &hashes,
	})
	require.Error(t, err)
}

func TestToDomainMeasurementNoAttestationPolicy(t *testing.T) {
	measurement, err := toDomainMeasurement(Measurement{
		Name:            "no-attestation",
		AttestationType: "none",
	})
	require.NoError(t, err)
	require.Empty(t, measurement.Measurement)

	_, err = toDomainMeasurement(Measurement{
		Name:            "invalid-no-attestation",
		AttestationType: "none",
		Measurements: map[string]domain.SingleMeasurement{
			"0": {Expected: "aaaa"},
		},
	})
	require.EqualError(t, err, "none attestation type must not contain measurements")
}

func validImageHashes() domain.DcapImageHashes {
	return domain.DcapImageHashes{
		UKIAuthenticode:    strings.Repeat("1", 96),
		KernelAuthenticode: strings.Repeat("2", 96),
		CmdlineHash:        strings.Repeat("3", 96),
		InitrdHash:         strings.Repeat("4", 96),
		GPTDiskGUIDHash:    strings.Repeat("5", 96),
	}
}
