package domain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSingleMeasurement_GetExpectedValues(t *testing.T) {
	t.Run("single expected value", func(t *testing.T) {
		sm := SingleMeasurement{Expected: "abc123"}
		require.Equal(t, []string{"abc123"}, sm.GetExpectedValues())
	})

	t.Run("multiple expected values via ExpectedAny", func(t *testing.T) {
		sm := SingleMeasurement{ExpectedAny: []string{"abc", "def", "ghi"}}
		require.Equal(t, []string{"abc", "def", "ghi"}, sm.GetExpectedValues())
	})

	t.Run("ExpectedAny takes precedence over Expected", func(t *testing.T) {
		sm := SingleMeasurement{
			Expected:    "single",
			ExpectedAny: []string{"multi1", "multi2"},
		}
		require.Equal(t, []string{"multi1", "multi2"}, sm.GetExpectedValues())
	})

	t.Run("empty returns nil", func(t *testing.T) {
		sm := SingleMeasurement{}
		require.Nil(t, sm.GetExpectedValues())
	})

	t.Run("empty ExpectedAny falls back to Expected", func(t *testing.T) {
		sm := SingleMeasurement{
			Expected:    "fallback",
			ExpectedAny: []string{},
		}
		require.Equal(t, []string{"fallback"}, sm.GetExpectedValues())
	})
}

func TestSingleMeasurement_JSON(t *testing.T) {
	t.Run("unmarshal single expected value", func(t *testing.T) {
		jsonData := `{"expected": "abc123"}`
		var sm SingleMeasurement
		err := json.Unmarshal([]byte(jsonData), &sm)
		require.NoError(t, err)
		require.Equal(t, "abc123", sm.Expected)
		require.Empty(t, sm.ExpectedAny)
	})

	t.Run("unmarshal expected_any array", func(t *testing.T) {
		jsonData := `{"expected_any": ["abc", "def", "ghi"]}`
		var sm SingleMeasurement
		err := json.Unmarshal([]byte(jsonData), &sm)
		require.NoError(t, err)
		require.Empty(t, sm.Expected)
		require.Equal(t, []string{"abc", "def", "ghi"}, sm.ExpectedAny)
	})

	t.Run("marshal single expected value", func(t *testing.T) {
		sm := SingleMeasurement{Expected: "abc123"}
		data, err := json.Marshal(sm)
		require.NoError(t, err)
		require.JSONEq(t, `{"expected": "abc123"}`, string(data))
	})

	t.Run("marshal expected_any array", func(t *testing.T) {
		sm := SingleMeasurement{ExpectedAny: []string{"abc", "def"}}
		data, err := json.Marshal(sm)
		require.NoError(t, err)
		require.JSONEq(t, `{"expected_any": ["abc", "def"]}`, string(data))
	})

	t.Run("omitempty works for empty fields", func(t *testing.T) {
		sm := SingleMeasurement{Expected: "value"}
		data, err := json.Marshal(sm)
		require.NoError(t, err)
		// Should not contain expected_any since it's empty
		require.NotContains(t, string(data), "expected_any")
	})
}

func TestMeasurement_JSON(t *testing.T) {
	t.Run("mixed single and expected_any values", func(t *testing.T) {
		jsonData := `{
			"8": {"expected": "0000"},
			"11": {"expected_any": ["aaaa", "bbbb"]},
			"4": {"expected": "cccc"}
		}`
		var m map[string]SingleMeasurement
		err := json.Unmarshal([]byte(jsonData), &m)
		require.NoError(t, err)
		require.Equal(t, []string{"0000"}, m["8"].GetExpectedValues())
		require.Equal(t, []string{"aaaa", "bbbb"}, m["11"].GetExpectedValues())
		require.Equal(t, []string{"cccc"}, m["4"].GetExpectedValues())
	})
}
