// Package application contains application logic for the builder-hub
package application

import (
	"encoding/json"
	"strings"

	"github.com/buger/jsonparser"
)

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
