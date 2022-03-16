package main

import (
	"log"
	"net"

	"github.com/Taluu/gabsee-test/generated/infrapb"
	"google.golang.org/grpc"
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
