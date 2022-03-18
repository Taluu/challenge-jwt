package secrets

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Taluu/gabsee-test/generated/infrapb"
	"github.com/golang-jwt/jwt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultSigningKey     = "colonel_gisberg"
)

// Service is the service that allow to interact with stored secrets through gRPC.
type Service struct {
	infrapb.UnimplementedSecretsServer

	store              SecretStore
	expirationDuration string
	signingKey         []byte
}

// NewService creates a new service with a given secrets store.
func NewService(store SecretStore) *Service {
	return &Service{
		store: store,

		expirationDuration: defaultExpirationDuration,
		signingKey:         []byte(defaultSigningKey),
	}
}

func (s *Service) List(ctx context.Context, in *infrapb.Empty) (*infrapb.SecretList, error) {
	secrets := make([]*infrapb.Secret, 0)
	storedSecrets, err := s.store.List(ctx)

	if err != nil {
		return &infrapb.SecretList{}, status.Errorf(codes.Internal, "couldn't retrieve list of secrets : %s", err)
	}

	for _, v := range storedSecrets {
		secrets = append(
			secrets,
			&infrapb.Secret{
				Name:   v.Name,
				Claims: v.Claims,
			},
		)
	}

	return &infrapb.SecretList{Secrets: secrets}, nil
}

func (s *Service) Delete(ctx context.Context, in *infrapb.Secret) (*infrapb.Empty, error) {
	err := s.store.Delete(ctx, in.Name)

	if err != nil {
		return &infrapb.Empty{}, status.Errorf(codes.Internal, "couldn't delete secret : %s", err)
	}

	return &infrapb.Empty{}, nil
}

func (s *Service) Create(ctx context.Context, in *infrapb.Secret) (*infrapb.Secret, error) {
	if contains, _ := s.store.Contains(ctx, in.Name); contains {
		return in, status.Errorf(codes.AlreadyExists, "secret name \"%s\" already exists", in.Name)
	}

	if in.Claims == nil {
		in.Claims = make(map[string]string)
	}

	if _, ok := in.Claims["exp"]; !ok {
		expirationDate, _ := time.ParseDuration(s.expirationDuration)
		in.Claims["exp"] = fmt.Sprint(time.Now().Add(expirationDate).Unix())
	}

	unix, err := strconv.Atoi(in.Claims["exp"])

	if err != nil {
		return in, status.Errorf(codes.InvalidArgument, "error when parsing time for the expiration date : %s", err)
	}

	expirationDate := time.Unix(int64(unix), 0)

	token, err := createToken(in.Name, in.Claims, s.signingKey)

	if err != nil {
		return in, status.Errorf(codes.Internal, "couldn't encode jwt: %s", err)
	}

	err = s.store.Save(
		ctx,
		Secret{
			Name:      in.Name,
			Claims:    in.Claims,
			ExpiresAt: expirationDate,
			Token:     token,
		},
	)

	if err != nil {
		return in, status.Errorf(codes.Internal, "couldn't create secret : %s", err)
	}

	return in, nil
}

func (s *Service) Update(ctx context.Context, in *infrapb.Secret) (*infrapb.Secret, error) {
	secret, err := s.store.Fetch(ctx, in.Name)

	if err != nil {
		return in, status.Errorf(codes.AlreadyExists, "secret name \"%s\" doesn't exists (%s)", in.Name, err)
	}

	if in.Claims == nil {
		in.Claims = make(map[string]string)
	}

	if _, ok := in.Claims["exp"]; !ok {
		expirationDate, _ := time.ParseDuration(s.expirationDuration)
		in.Claims["exp"] = fmt.Sprint(time.Now().Add(expirationDate).Unix())
	}

	expirationDate, err := strconv.Atoi(in.Claims["exp"])

	if err != nil {
		return in, status.Errorf(codes.InvalidArgument, "error when parsing time for the expiration date : %s", err)
	}

	for k, v := range in.Claims {
		secret.Claims[k] = v
	}

	secret.ExpiresAt = time.Unix(int64(expirationDate), 0)

	token, err := createToken(in.Name, in.Claims, s.signingKey)

	if err != nil {
		return in, status.Errorf(codes.Internal, "couldn't encode jwt: %s", err)
	}

	secret.Token = token

	s.store.Save(ctx, secret)

	return in, nil
}

func createToken(name string, claims map[string]string, signingKey []byte) (string, error) {
	tokenClaims := jwt.MapClaims{}
	tokenClaims["id"] = name

	for k, v := range claims {
		tokenClaims[k] = v
	}

	// overwrite the exp to be a number rather than a string
	tokenClaims["exp"], _ = strconv.Atoi(claims["exp"])

	return jwt.
		NewWithClaims(jwt.SigningMethodHS256, tokenClaims).
		SignedString(signingKey)
}
