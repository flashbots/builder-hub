// Package secrets contains logic for adapter to aws secrets manager
package secrets

import (
	"encoding/json"
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

type Service struct {
	sm           *secretsmanager.SecretsManager
	secretPrefix string
}

func NewService(secretPrefix string) (*Service, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-2"),
	})
	if err != nil {
		return nil, err
	}

	// Create a Secrets Manager client
	svc := secretsmanager.New(sess)

	return &Service{sm: svc, secretPrefix: secretPrefix}, nil
}

var ErrMissingSecret = errors.New("missing secret for builder")

func (s *Service) secretName(builderName string) string {
	return s.secretPrefix + "_" + builderName
}

func (s *Service) GetSecretValues(builderName string) (json.RawMessage, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(s.secretName(builderName)),
	}

	result, err := s.sm.GetSecretValue(input)
	if err != nil {
		// If the secret doesn't exist, return empty JSON for new builders
		var awsErr awserr.Error
		if errors.As(err, &awsErr) && awsErr.Code() == secretsmanager.ErrCodeResourceNotFoundException {
			return json.RawMessage("{}"), nil
		}
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
	secretName := s.secretName(builderName)
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}

	result, err := s.sm.GetSecretValue(input)
	var secretData map[string]json.RawMessage

	if err != nil {
		// If the secret doesn't exist, create it
		var awsErr awserr.Error
		if errors.As(err, &awsErr) && awsErr.Code() == secretsmanager.ErrCodeResourceNotFoundException {
			// Create a new secret with the builder's values
			secretData = map[string]json.RawMessage{builderName: values}
			newSecretString, marshalErr := json.Marshal(secretData)
			if marshalErr != nil {
				return marshalErr
			}

			createInput := &secretsmanager.CreateSecretInput{
				Name:         aws.String(secretName),
				SecretString: aws.String(string(newSecretString)),
			}
			_, createErr := s.sm.CreateSecret(createInput)
			return createErr
		}
		return err
	}

	// Secret exists, update it
	secretData = make(map[string]json.RawMessage)
	err = json.Unmarshal([]byte(*result.SecretString), &secretData)
	if err != nil {
		return err
	}

	secretData[builderName] = values
	newSecretString, err := json.Marshal(secretData)
	if err != nil {
		return err
	}

	sv := &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(secretName),
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
