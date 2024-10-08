package secrets

import (
	"encoding/json"

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

func (s *Service) GetSecretValues(builderName string) (map[string]string, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(s.secretName),
	}

	result, err := s.sm.GetSecretValue(input)
	if err != nil {
		return nil, err
	}
	secretData := make(map[string]string)
	err = json.Unmarshal([]byte(*result.SecretString), &secretData)
	if err != nil {
		return nil, err
	}
	defaultStr := secretData["default"]
	defaultSecrets := make(map[string]string)
	err = json.Unmarshal([]byte(defaultStr), &defaultSecrets)
	if err != nil {
		return nil, err
	}

	builderStr, ok := secretData[builderName]
	if !ok {
		return defaultSecrets, nil
	}
	builderSecrets := make(map[string]string)
	err = json.Unmarshal([]byte(builderStr), &builderSecrets)
	if err != nil {
		return nil, err
	}
	return MergeSecrets(defaultSecrets, builderSecrets), nil
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
