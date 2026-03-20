package domain

import (
	"context"
	"encoding/json"
	"sync"
)

type InmemorySecretService struct {
	mu *sync.RWMutex
	st map[string]json.RawMessage
}

func NewMockSecretService() *InmemorySecretService {
	return &InmemorySecretService{
		st: make(map[string]json.RawMessage),
		mu: &sync.RWMutex{},
	}
}

func (mss *InmemorySecretService) GetSecretValues(ctx context.Context, builderName string) (json.RawMessage, error) {
	mss.mu.RLock()
	defer mss.mu.RUnlock()
	return mss.st[builderName], nil
}

func (mss *InmemorySecretService) SetSecretValues(ctx context.Context, builderName string, values json.RawMessage) error {
	mss.mu.Lock()
	defer mss.mu.Unlock()
	mss.st[builderName] = values
	return nil
}
