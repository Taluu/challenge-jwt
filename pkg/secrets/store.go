package secrets

import (
	"context"
	"fmt"
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
	Contains(context.Context, string) (bool, error)
	Fetch(context.Context, string) (Secret, error)
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

func (s *secretStore) Contains(ctx context.Context, name string) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	_, exists := s.secrets[name]

	return exists, nil
}

func (s *secretStore) Fetch(ctx context.Context, name string) (Secret, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	secret, exists := s.secrets[name]

	if !exists {
		return Secret{}, fmt.Errorf("no such secret")
	}

	return secret, nil
}
