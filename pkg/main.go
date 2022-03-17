package main

import (
	"log"
	"net"

	"github.com/Taluu/gabsee-test/generated/infrapb"
	"github.com/Taluu/gabsee-test/pkg/secrets"
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
		secrets.NewService(
			secrets.NewSecretStore(),
			secrets.Config{},
		),
	)

	log.Fatalln(server.Serve(listener))
}
