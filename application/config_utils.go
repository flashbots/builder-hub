// Package application contains application logic for the builder-hub
package application

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/buger/jsonparser"
)

var ErrNonStringSecret = errors.New("secret value is not a string")

func MergeConfigSecrets(config json.RawMessage, secrets map[string]string) (json.RawMessage, error) {
	// merge config and secrets
	bts := []byte(config)
	var err error
	for k, v := range secrets {
		tV := "\"" + v + "\""
		bts, err = jsonparser.Set(bts, []byte(tV), strings.Split(k, ".")...)
		if err != nil {
			return nil, err
		}

	}
	return bts, nil
}

// Recursive function to flatten JSON objects
func flattenJSON(data map[string]interface{}, prefix string, flatMap map[string]string) error {
	for key, value := range data {
		newKey := key
		if prefix != "" {
			newKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			err := flattenJSON(v, newKey, flatMap)
			if err != nil {
				return err
			}
		case []interface{}:
			err := flattenArray(v, newKey, flatMap)
			if err != nil {
				return err
			}
		case string:
			flatMap[newKey] = v
		default:
			// skipping
			return ErrNonStringSecret
		}
	}
	return nil
}

// Recursive function to flatten JSON arrays
func flattenArray(data []interface{}, prefix string, flatMap map[string]string) error {
	for i, value := range data {
		newKey := prefix + "." + "[" + strconv.Itoa(i) + "]"
		switch v := value.(type) {
		case map[string]interface{}:
			err := flattenJSON(v, newKey, flatMap)
			if err != nil {
				return err
			}
		case []interface{}:
			err := flattenArray(v, newKey, flatMap)
			if err != nil {
				return err
			}
		case string:
			flatMap[newKey] = v
		default:
			return ErrNonStringSecret
		}
	}
	return nil
}

// FlattenJSONFromBytes is the main function to process JSON from []byte input
func FlattenJSONFromBytes(jsonBytes []byte) (map[string]string, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return nil, err
	}

	flatMap := make(map[string]string)
	err := flattenJSON(data, "", flatMap)
	if err != nil {
		return nil, err
	}
	return flatMap, nil
}
