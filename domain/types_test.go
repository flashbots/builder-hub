package domain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSingleMeasurement_UnmarshalJSON(t *testing.T) {
	t.Run("single string value (backwards compatible)", func(t *testing.T) {
		jsonData := `{"expected": "abc123"}`
		var sm SingleMeasurement
		err := json.Unmarshal([]byte(jsonData), &sm)
		require.NoError(t, err)
		require.Equal(t, []string{"abc123"}, sm.Expected)
	})

	t.Run("array of strings", func(t *testing.T) {
		jsonData := `{"expected": ["abc123", "def456", "ghi789"]}`
		var sm SingleMeasurement
		err := json.Unmarshal([]byte(jsonData), &sm)
		require.NoError(t, err)
		require.Equal(t, []string{"abc123", "def456", "ghi789"}, sm.Expected)
	})

	t.Run("empty array", func(t *testing.T) {
		jsonData := `{"expected": []}`
		var sm SingleMeasurement
		err := json.Unmarshal([]byte(jsonData), &sm)
		require.NoError(t, err)
		require.Equal(t, []string{}, sm.Expected)
	})

	t.Run("single element array", func(t *testing.T) {
		jsonData := `{"expected": ["only_one"]}`
		var sm SingleMeasurement
		err := json.Unmarshal([]byte(jsonData), &sm)
		require.NoError(t, err)
		require.Equal(t, []string{"only_one"}, sm.Expected)
	})
}

func TestSingleMeasurement_MarshalJSON(t *testing.T) {
	t.Run("single value marshals as string (backwards compatible)", func(t *testing.T) {
		sm := SingleMeasurement{Expected: []string{"abc123"}}
		data, err := json.Marshal(sm)
		require.NoError(t, err)
		require.JSONEq(t, `{"expected": "abc123"}`, string(data))
	})

	t.Run("multiple values marshal as array", func(t *testing.T) {
		sm := SingleMeasurement{Expected: []string{"abc123", "def456"}}
		data, err := json.Marshal(sm)
		require.NoError(t, err)
		require.JSONEq(t, `{"expected": ["abc123", "def456"]}`, string(data))
	})

	t.Run("empty array marshals as array", func(t *testing.T) {
		sm := SingleMeasurement{Expected: []string{}}
		data, err := json.Marshal(sm)
		require.NoError(t, err)
		require.JSONEq(t, `{"expected": []}`, string(data))
	})
}

func TestSingleMeasurement_RoundTrip(t *testing.T) {
	t.Run("single string round trip", func(t *testing.T) {
		original := `{"expected": "abc123"}`
		var sm SingleMeasurement
		err := json.Unmarshal([]byte(original), &sm)
		require.NoError(t, err)

		data, err := json.Marshal(sm)
		require.NoError(t, err)
		require.JSONEq(t, original, string(data))
	})

	t.Run("array round trip", func(t *testing.T) {
		original := SingleMeasurement{Expected: []string{"abc", "def", "ghi"}}
		data, err := json.Marshal(original)
		require.NoError(t, err)

		var sm SingleMeasurement
		err = json.Unmarshal(data, &sm)
		require.NoError(t, err)
		require.Equal(t, original.Expected, sm.Expected)
	})
}

func TestMeasurement_UnmarshalJSON(t *testing.T) {
	t.Run("mixed single and array values", func(t *testing.T) {
		jsonData := `{
			"8": {"expected": "0000"},
			"11": {"expected": ["aaaa", "bbbb"]},
			"4": {"expected": "cccc"}
		}`
		var m map[string]SingleMeasurement
		err := json.Unmarshal([]byte(jsonData), &m)
		require.NoError(t, err)
		require.Equal(t, []string{"0000"}, m["8"].Expected)
		require.Equal(t, []string{"aaaa", "bbbb"}, m["11"].Expected)
		require.Equal(t, []string{"cccc"}, m["4"].Expected)
	})
}
