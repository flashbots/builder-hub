package domain

import "sync"

type InmemorySecretService struct {
	mu *sync.RWMutex
	st map[string]map[string]string
}

func NewMockSecretService() *InmemorySecretService {
	return &InmemorySecretService{
		st: make(map[string]map[string]string),
		mu: &sync.RWMutex{},
	}
}

func (mss *InmemorySecretService) GetSecretValues(builderName string) (map[string]string, error) {
	mss.mu.RLock()
	defer mss.mu.RUnlock()
	return mss.st[builderName], nil
}

func (mss *InmemorySecretService) SetSecretValues(builderName string, values map[string]string) error {
	mss.mu.Lock()
	defer mss.mu.Unlock()
	mss.st[builderName] = values
	return nil
}
