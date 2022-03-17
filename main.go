package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/Taluu/gabsee-test/generated/infrapb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func main() {
	log.Println("Server running ...")

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalln(err)
	}

	server := grpc.NewServer()

	infrapb.RegisterSecretsServer(
		server,
		NewService(
			NewSecretStore(),
		),
	)

	log.Fatalln(server.Serve(listener))
}

const (
	defaultExpirationDuration = "86400s" // +1 day
)

// Service is the service that allow to interact with stored secrets through gRPC.
type Service struct {
	infrapb.UnimplementedSecretsServer

	store SecretStore
}

// NewService creates a new service with a given secrets store.
func NewService(store SecretStore) *Service {
	return &Service{
		store: store,
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
		expirationDate, _ := time.ParseDuration(defaultExpirationDuration)
		in.Claims["exp"] = fmt.Sprint(time.Now().Add(expirationDate).Unix())
	}

	unix, err := strconv.Atoi(in.Claims["exp"])

	if err != nil {
		return in, status.Errorf(codes.InvalidArgument, "error when parsing time for the expiration date : %s", err)
	}

	expirationDate := time.Unix(int64(unix), 0)

	//TODO: generate jwt token

	err = s.store.Save(
		ctx,
		Secret{
			Name:      in.Name,
			Claims:    in.Claims,
			ExpiresAt: expirationDate,
		},
	)

	if err != nil {
		return in, status.Errorf(codes.Internal, "couldn't create secret : %s", err)
	}

	return in, nil
}
