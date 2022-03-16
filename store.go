package main

import (
	"context"
	"sync"
	"time"
)

type Secret struct {
	Name      string
	Token     string
	ExpiresAt time.Time
	Claims    map[string]string
}

type secretStore struct {
	secrets map[string]Secret
	lock    sync.Mutex
}

type SecretStore interface {
	Save(context.Context, Secret) error
	Delete(context.Context, string) error
	List(context.Context) ([]Secret, error)
}

func NewSecretStore() SecretStore {
	return &secretStore{
		secrets: make(map[string]Secret),
	}
}

func (s *secretStore) Save(ctx context.Context, in Secret) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.secrets[in.Name] = in

	return nil
}

func (s *secretStore) Delete(ctx context.Context, name string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.secrets, name)

	return nil
}

func (s *secretStore) List(ctx context.Context) ([]Secret, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	var result = make([]Secret, 0, len(s.secrets))

	for _, v := range s.secrets {
		result = append(result, v)
	}

	return result, nil
}
