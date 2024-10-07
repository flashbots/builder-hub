package application

import (
	"encoding/json"
	"strings"

	"github.com/buger/jsonparser"
)

func MergeConfigSecrets(config json.RawMessage, secrets map[string]string) (json.RawMessage, error) {
	// merge config and secrets
	var bts = []byte(config)
	var err error
	for k, v := range secrets {
		tV := "\"" + v + "\""
		bts, err = jsonparser.Set(config, []byte(tV), strings.Split(k, ".")...)
		if err != nil {
			return nil, err
		}

	}
	return bts, nil
}

func MergeSecrets(defaultSecrets map[string]string, secrets map[string]string) map[string]string {
	// merge secrets
	res := make(map[string]string)
	for k, v := range defaultSecrets {
		res[k] = v
	}
	for k, v := range secrets {
		res[k] = v
	}
	return res
}
