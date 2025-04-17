// Package secrets contains logic for adapter to aws secrets manager
package secrets

import (
	"encoding/json"
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

type Service struct {
	sm         *secretsmanager.SecretsManager
	secretName string
}

func NewService(secretName string) (*Service, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-2"),
	})
	if err != nil {
		return nil, err
	}

	// Create a Secrets Manager client
	svc := secretsmanager.New(sess)

	return &Service{sm: svc, secretName: secretName}, nil
}

var ErrMissingSecret = errors.New("missing secret for builder")

func (s *Service) GetSecretValues(builderName string) (json.RawMessage, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(s.secretName),
	}

	result, err := s.sm.GetSecretValue(input)
	if err != nil {
		return nil, err
	}
	secretData := make(map[string]json.RawMessage)
	err = json.Unmarshal([]byte(*result.SecretString), &secretData)
	if err != nil {
		return nil, err
	}

	builderSecret, ok := secretData[builderName]
	if !ok {
		return nil, ErrMissingSecret
	}

	return builderSecret, nil
}

func (s *Service) SetSecretValues(builderName string, values json.RawMessage) error {
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(s.secretName),
	}

	result, err := s.sm.GetSecretValue(input)
	if err != nil {
		return err
	}
	secretData := make(map[string]string)
	err = json.Unmarshal([]byte(*result.SecretString), &secretData)
	if err != nil {
		return err
	}

	marshalValues, err := json.Marshal(values)
	if err != nil {
		return err
	}
	secretData[builderName] = string(marshalValues)
	newSecretString, err := json.Marshal(secretData)
	if err != nil {
		return err
	}

	sv := &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(s.secretName),
		SecretString: aws.String(string(newSecretString)),
	}
	_, err = s.sm.PutSecretValue(sv)
	if err != nil {
		return err
	}
	return nil
}

func MergeSecrets(defaultSecrets, secrets map[string]string) map[string]string {
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
