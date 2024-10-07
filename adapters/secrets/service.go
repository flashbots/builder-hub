package secrets

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

type Service struct {
	sm *secretsmanager.SecretsManager
}

func NewService() (*Service, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-2"),
	})
	if err != nil {
		return nil, err
	}

	// Create a Secrets Manager client
	svc := secretsmanager.New(sess)

	return &Service{sm: svc}, nil
}

func (s *Service) GetSecretValues(secretName string) (map[string]string, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
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
	return secretData, nil
}
