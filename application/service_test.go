package application

import (
	"strings"
	"testing"

	"github.com/flashbots/builder-hub/domain"
	"github.com/stretchr/testify/require"
)

func TestCheckMeasurement_SingleExpectedValue(t *testing.T) {
	template := domain.Measurement{
		Name:            "test",
		AttestationType: "azure-tdx",
		Measurement: map[string]domain.SingleMeasurement{
			"8":  {Expected: "0000"},
			"11": {Expected: "aaaa"},
		},
	}

	t.Run("exact match", func(t *testing.T) {
		measurement := map[string]string{
			"8":  "0000",
			"11": "aaaa",
		}
		require.True(t, checkMeasurement(measurement, template))
	})

	t.Run("wrong value", func(t *testing.T) {
		measurement := map[string]string{
			"8":  "0000",
			"11": "bbbb",
		}
		require.False(t, checkMeasurement(measurement, template))
	})

	t.Run("missing key", func(t *testing.T) {
		measurement := map[string]string{
			"8": "0000",
		}
		require.False(t, checkMeasurement(measurement, template))
	})

	t.Run("extra key in measurement is allowed", func(t *testing.T) {
		measurement := map[string]string{
			"8":  "0000",
			"11": "aaaa",
			"99": "extra",
		}
		require.True(t, checkMeasurement(measurement, template))
	})
}

func TestCheckMeasurement_MultipleExpectedValues(t *testing.T) {
	template := domain.Measurement{
		Name:            "test",
		AttestationType: "azure-tdx",
		Measurement: map[string]domain.SingleMeasurement{
			"8":  {ExpectedAny: []string{"0000", "1111", "2222"}},
			"11": {ExpectedAny: []string{"aaaa", "bbbb"}},
		},
	}

	t.Run("first expected value matches", func(t *testing.T) {
		measurement := map[string]string{
			"8":  "0000",
			"11": "aaaa",
		}
		require.True(t, checkMeasurement(measurement, template))
	})

	t.Run("second expected value matches", func(t *testing.T) {
		measurement := map[string]string{
			"8":  "1111",
			"11": "bbbb",
		}
		require.True(t, checkMeasurement(measurement, template))
	})

	t.Run("third expected value for key 8", func(t *testing.T) {
		measurement := map[string]string{
			"8":  "2222",
			"11": "aaaa",
		}
		require.True(t, checkMeasurement(measurement, template))
	})

	t.Run("value not in expected list", func(t *testing.T) {
		measurement := map[string]string{
			"8":  "3333",
			"11": "aaaa",
		}
		require.False(t, checkMeasurement(measurement, template))
	})

	t.Run("one key matches, other does not", func(t *testing.T) {
		measurement := map[string]string{
			"8":  "0000",
			"11": "cccc",
		}
		require.False(t, checkMeasurement(measurement, template))
	})
}

func TestValidateMeasurement(t *testing.T) {
	templates := []domain.Measurement{
		{
			Name:            "template-1",
			AttestationType: "azure-tdx",
			Measurement: map[string]domain.SingleMeasurement{
				"8":  {Expected: "0000"},
				"11": {ExpectedAny: []string{"aaaa", "bbbb"}},
			},
		},
		{
			Name:            "template-2",
			AttestationType: "azure-tdx",
			Measurement: map[string]domain.SingleMeasurement{
				"8":  {Expected: "1111"},
				"11": {Expected: "cccc"},
			},
		},
	}

	t.Run("matches first template", func(t *testing.T) {
		measurement := map[string]string{
			"8":  "0000",
			"11": "aaaa",
		}
		name, err := validateMeasurement(legacyMeasurements(measurement), templates)
		require.NoError(t, err)
		require.Equal(t, "template-1", name)
	})

	t.Run("matches first template with second expected value", func(t *testing.T) {
		measurement := map[string]string{
			"8":  "0000",
			"11": "bbbb",
		}
		name, err := validateMeasurement(legacyMeasurements(measurement), templates)
		require.NoError(t, err)
		require.Equal(t, "template-1", name)
	})

	t.Run("matches second template", func(t *testing.T) {
		measurement := map[string]string{
			"8":  "1111",
			"11": "cccc",
		}
		name, err := validateMeasurement(legacyMeasurements(measurement), templates)
		require.NoError(t, err)
		require.Equal(t, "template-2", name)
	})

	t.Run("no match returns error", func(t *testing.T) {
		measurement := map[string]string{
			"8":  "9999",
			"11": "zzzz",
		}
		_, err := validateMeasurement(legacyMeasurements(measurement), templates)
		require.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("legacy header does not match image policy", func(t *testing.T) {
		hashes := testImageHashes("a")
		imageTemplates := []domain.Measurement{
			{
				Name:            "portable-image",
				AttestationType: "dcap-tdx",
				DcapImageHashes: &hashes,
			},
		}

		_, err := validateMeasurement(legacyMeasurements(map[string]string{
			"0": strings.Repeat("a", 96),
		}), imageTemplates)
		require.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestCheckExpectedMeasurementRegisters(t *testing.T) {
	template := domain.Measurement{
		Name:            "azure-policy",
		AttestationType: "azure-tdx",
		Measurement: map[string]domain.SingleMeasurement{
			"pcr4":  {ExpectedAny: []string{"AAAA", "bbbb"}},
			"PCR11": {Expected: "CCCC"},
		},
	}

	t.Run("matches exact normalized policy", func(t *testing.T) {
		supplied := domain.SuppliedMeasurements{
			Type: domain.ExpectedMeasurementAzure,
			Registers: map[string][]string{
				"4":  {"BBBB", "aaaa"},
				"11": {"cccc"},
			},
		}
		require.True(t, checkExpectedMeasurement(supplied, template))
	})

	t.Run("rejects missing alternative", func(t *testing.T) {
		supplied := domain.SuppliedMeasurements{
			Type: domain.ExpectedMeasurementAzure,
			Registers: map[string][]string{
				"4":  {"aaaa"},
				"11": {"cccc"},
			},
		}
		require.False(t, checkExpectedMeasurement(supplied, template))
	})

	t.Run("rejects additional register", func(t *testing.T) {
		supplied := domain.SuppliedMeasurements{
			Type: domain.ExpectedMeasurementAzure,
			Registers: map[string][]string{
				"4":  {"aaaa", "bbbb"},
				"11": {"cccc"},
				"12": {"dddd"},
			},
		}
		require.False(t, checkExpectedMeasurement(supplied, template))
	})

	t.Run("matches DCAP aliases", func(t *testing.T) {
		dcapTemplate := domain.Measurement{
			AttestationType: "dcap-tdx",
			Measurement: map[string]domain.SingleMeasurement{
				"MRTD":  {Expected: "aaaa"},
				"rtmr2": {ExpectedAny: []string{"bbbb", "cccc"}},
			},
		}
		supplied := domain.SuppliedMeasurements{
			Type: domain.ExpectedMeasurementDCAP,
			Registers: map[string][]string{
				"0": {"AAAA"},
				"3": {"cccc", "bbbb"},
			},
		}
		require.True(t, checkExpectedMeasurement(supplied, dcapTemplate))
	})
}

func TestCheckExpectedMeasurementImage(t *testing.T) {
	hashes := testImageHashes("a")
	supplied := domain.SuppliedMeasurements{
		Type:            domain.ExpectedMeasurementImage,
		DcapImageHashes: &hashes,
	}

	for _, attestationType := range []string{"dcap-tdx", "gcp-tdx"} {
		t.Run(attestationType, func(t *testing.T) {
			templateHashes := testImageHashes("A")
			template := domain.Measurement{
				AttestationType: attestationType,
				DcapImageHashes: &templateHashes,
			}
			require.True(t, checkExpectedMeasurement(supplied, template))

			template.DcapImageHashes.InitrdHash = strings.Repeat("b", 96)
			require.False(t, checkExpectedMeasurement(supplied, template))
		})
	}
}

func TestCheckExpectedMeasurementAllowAnyAndNoAttestation(t *testing.T) {
	require.True(t, checkExpectedMeasurement(domain.SuppliedMeasurements{
		Type:      domain.ExpectedMeasurementDCAP,
		Registers: map[string][]string{},
	}, domain.Measurement{
		AttestationType: "gcp-tdx",
		Measurement:     map[string]domain.SingleMeasurement{},
	}))

	require.True(t, checkExpectedMeasurement(domain.SuppliedMeasurements{
		Type: domain.ExpectedMeasurementNoAttestation,
	}, domain.Measurement{
		AttestationType: "none",
		Measurement:     map[string]domain.SingleMeasurement{},
	}))

	require.False(t, checkExpectedMeasurement(domain.SuppliedMeasurements{
		Type: domain.ExpectedMeasurementNoAttestation,
	}, domain.Measurement{
		AttestationType: "none",
		Measurement: map[string]domain.SingleMeasurement{
			"0": {Expected: "aaaa"},
		},
	}))

	require.False(t, checkExpectedMeasurement(domain.SuppliedMeasurements{
		Type: domain.ExpectedMeasurementNoAttestation,
	}, domain.Measurement{AttestationType: "none"}))

	require.False(t, checkExpectedMeasurement(domain.SuppliedMeasurements{
		Type:      domain.ExpectedMeasurementDCAP,
		Registers: map[string][]string{},
	}, domain.Measurement{AttestationType: "dcap-tdx"}))
}

func legacyMeasurements(values map[string]string) domain.SuppliedMeasurements {
	return domain.SuppliedMeasurements{
		Type:   domain.ExpectedMeasurementLegacy,
		Legacy: values,
	}
}

func testImageHashes(character string) domain.DcapImageHashes {
	value := strings.Repeat(character, 96)
	return domain.DcapImageHashes{
		UKIAuthenticode:    value,
		KernelAuthenticode: value,
		CmdlineHash:        value,
		InitrdHash:         value,
		GPTDiskGUIDHash:    value,
	}
}

func TestMatchesAnyExpected(t *testing.T) {
	t.Run("empty list returns false", func(t *testing.T) {
		require.False(t, matchesAnyExpected("value", []string{}))
	})

	t.Run("single value match", func(t *testing.T) {
		require.True(t, matchesAnyExpected("value", []string{"value"}))
	})

	t.Run("single value no match", func(t *testing.T) {
		require.False(t, matchesAnyExpected("value", []string{"other"}))
	})

	t.Run("multiple values first match", func(t *testing.T) {
		require.True(t, matchesAnyExpected("a", []string{"a", "b", "c"}))
	})

	t.Run("multiple values middle match", func(t *testing.T) {
		require.True(t, matchesAnyExpected("b", []string{"a", "b", "c"}))
	})

	t.Run("multiple values last match", func(t *testing.T) {
		require.True(t, matchesAnyExpected("c", []string{"a", "b", "c"}))
	})

	t.Run("multiple values no match", func(t *testing.T) {
		require.False(t, matchesAnyExpected("d", []string{"a", "b", "c"}))
	})
}
