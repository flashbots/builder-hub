package application

import (
	"testing"

	"github.com/flashbots/builder-hub/domain"
	"github.com/stretchr/testify/require"
)

func TestCheckMeasurement_SingleExpectedValue(t *testing.T) {
	template := domain.Measurement{
		Name:            "test",
		AttestationType: "azure-tdx",
		Measurement: map[string]domain.SingleMeasurement{
			"8":  {Expected: []string{"0000"}},
			"11": {Expected: []string{"aaaa"}},
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
			"8":  {Expected: []string{"0000", "1111", "2222"}},
			"11": {Expected: []string{"aaaa", "bbbb"}},
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
				"8":  {Expected: []string{"0000"}},
				"11": {Expected: []string{"aaaa", "bbbb"}},
			},
		},
		{
			Name:            "template-2",
			AttestationType: "azure-tdx",
			Measurement: map[string]domain.SingleMeasurement{
				"8":  {Expected: []string{"1111"}},
				"11": {Expected: []string{"cccc"}},
			},
		},
	}

	t.Run("matches first template", func(t *testing.T) {
		measurement := map[string]string{
			"8":  "0000",
			"11": "aaaa",
		}
		name, err := validateMeasurement(measurement, templates)
		require.NoError(t, err)
		require.Equal(t, "template-1", name)
	})

	t.Run("matches first template with second expected value", func(t *testing.T) {
		measurement := map[string]string{
			"8":  "0000",
			"11": "bbbb",
		}
		name, err := validateMeasurement(measurement, templates)
		require.NoError(t, err)
		require.Equal(t, "template-1", name)
	})

	t.Run("matches second template", func(t *testing.T) {
		measurement := map[string]string{
			"8":  "1111",
			"11": "cccc",
		}
		name, err := validateMeasurement(measurement, templates)
		require.NoError(t, err)
		require.Equal(t, "template-2", name)
	})

	t.Run("no match returns error", func(t *testing.T) {
		measurement := map[string]string{
			"8":  "9999",
			"11": "zzzz",
		}
		_, err := validateMeasurement(measurement, templates)
		require.ErrorIs(t, err, domain.ErrNotFound)
	})
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
