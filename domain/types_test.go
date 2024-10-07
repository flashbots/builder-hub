package domain

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalculateHash(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		measurements := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}
		hash := CalculateHash(measurements)
		require.Equal(t, fmt.Sprintf("%x", hash), "b734413c644ec49f6a7c07d88b267244582d6422d89eee955511f6b3c0dcb0f2")
	})
	t.Run("unordered", func(t *testing.T) {
		measurements := map[string]string{
			"1": "2214214214214",
			"3": "217429174987412",
			"2": "2124214",
		}
		hash := CalculateHash(measurements)
		require.Equal(t, fmt.Sprintf("%x", hash), "4e5f14b1f6f1a1cbff43a08b2a2f76b261e98fa8c43b51c57b120923a5f10462")
	})

}
