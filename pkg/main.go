package main

import (
	"log"
	"net"
	"os"
	"time"

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

	config := secrets.Config{}

	if TTL, ok := os.LookupEnv("SECRETS_TTL"); ok {
		config.TTL, err = time.ParseDuration(TTL)

		if err != nil {
			log.Fatalln(err)
		}
	}

	if NearTTL, ok := os.LookupEnv("SECRETS_NEAR_TTL"); ok {
		config.NearTTL, err = time.ParseDuration(NearTTL)

		if err != nil {
			log.Fatalln(err)
		}
	}

	if tickDuration, ok := os.LookupEnv("SECRETS_TICK"); ok {
		config.TickDuration, err = time.ParseDuration(tickDuration)

		if err != nil {
			log.Fatalln(err)
		}
	}

	if signingKey, ok := os.LookupEnv("SECRETS_JWT_SIGNING_KEY"); ok {
		config.SigningKey = []byte(signingKey)
	}

	server := grpc.NewServer()
	service := secrets.NewService(secrets.NewSecretStore(), config)

	infrapb.RegisterSecretsServer(server, service)

	log.Fatalln(server.Serve(listener))
}
