package domain

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalculateHash(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		measurements := map[string]SingleMeasurement{
			"key1": {"value1"},
			"key2": {"value2"},
		}
		hash := CalculateHash(measurements)
		require.Equal(t, fmt.Sprintf("%x", hash), "f0605189145d385e22a5dcba5f850d6fcafe83dde8586bc79b8f2dc3450ae849")
	})
	t.Run("unordered", func(t *testing.T) {
		measurements := map[string]SingleMeasurement{
			"1": {"2214214214214"},
			"3": {"217429174987412"},
			"2": {"2124214"},
		}
		hash := CalculateHash(measurements)
		require.Equal(t, fmt.Sprintf("%x", hash), "2a4d79238e4876617c64eb3d5a46f8e851399d0f91311b8d74e2b499fd2b576d")
	})
	t.Run("attestation-td", func(t *testing.T) {
		measuremnt := make(map[string]SingleMeasurement)
		s := `{
	"11": {
		"expected": "0000000000000000000000000000000000000000000000000000000000000000"
	},
	"12": {
		"expected": "f1a142c53586e7e2223ec74e5f4d1a4942956b1fd9ac78fafcdf85117aa345da"
	},
	"13": {
		"expected": "0000000000000000000000000000000000000000000000000000000000000000"
	},
	"15": {
		"expected": "0000000000000000000000000000000000000000000000000000000000000000"
	},
	"4": {
		"expected": "98ba2c602b62e67b8e0bd6c6676f12ade320a763e5e4564f62fd875a502dd651"
	},
	"8": {
		"expected": "0000000000000000000000000000000000000000000000000000000000000000"
	},
	"9": {
		"expected": "e77938394412d83a8d4de52cdaf97df82a4d4059e1e7c4fc3c73581816cea496"
	}
}`
		err := json.Unmarshal([]byte(s), &measuremnt)
		require.NoError(t, err)
		hash := CalculateHash(measuremnt)
		require.Equal(t, fmt.Sprintf("%x", hash), "538c7595f3570c9dab335a0232c84a7e7a352acd4c84f78cd2b5e70429acf30b")
	})

}
