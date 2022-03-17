package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/Taluu/gabsee-test/generated/infrapb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

type testConnection struct {
	conn   *bufconn.Listener
	t      *testing.T
	wg     sync.WaitGroup
	server *grpc.Server
}

func newTestConnection(t *testing.T, store SecretStore) *testConnection {
	c := testConnection{
		conn:   bufconn.Listen(1024 * 1024),
		t:      t,
		server: grpc.NewServer(),
	}

	infrapb.RegisterSecretsServer(c.server, NewService(store))

	return &c
}

func (c *testConnection) Start() {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		if err := c.server.Serve(c.conn); err != nil {
			log.Fatal("GRPC server exited with error:", err)
		}
	}()
}

func (c *testConnection) Stop() {
	if c.server == nil {
		return
	}
	c.server.Stop()
	c.wg.Wait()
}

func (c *testConnection) Dial(ctx context.Context) *grpc.ClientConn {
	conn, err := grpc.DialContext(
		ctx, "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return c.conn.Dial()
		}),
		grpc.WithInsecure(),
	)
	if err != nil {
		c.t.Fatalf("Couldn't connect: %v", err)
	}
	return conn
}

func newTestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

func TestList(t *testing.T) {
	store := NewSecretStore()
	conn := newTestConnection(t, store)

	conn.Start()
	defer conn.Stop()

	ctx, cancel := newTestContext()
	defer cancel()

	client := infrapb.NewSecretsClient(conn.Dial(ctx))
	_, err := client.List(ctx, &infrapb.Empty{})

	if err != nil {
		t.Fatalf("Unexpected error : %e", err)
	}
}

func TestDelete(t *testing.T) {
	store := NewSecretStore()
	conn := newTestConnection(t, store)

	conn.Start()
	defer conn.Stop()

	ctx, cancel := newTestContext()
	defer cancel()

	// fixture
	store.Save(ctx, Secret{
		Name: "foo",
	})

	client := infrapb.NewSecretsClient(conn.Dial(ctx))
	_, err := client.Delete(
		ctx,
		&infrapb.Secret{
			Name: "foo",
		},
	)

	if err != nil {
		t.Fatalf("Unexpected error : %e", err)
	}

	if contains, _ := store.Contains(ctx, "foo"); contains {
		t.Fatalf("Secret not deleted")
	}
}

func TestDeleteUnknownSecret(t *testing.T) {
	store := NewSecretStore()
	conn := newTestConnection(t, store)

	conn.Start()
	defer conn.Stop()

	ctx, cancel := newTestContext()
	defer cancel()

	client := infrapb.NewSecretsClient(conn.Dial(ctx))
	_, err := client.Delete(
		ctx,
		&infrapb.Secret{
			Name: "foo",
		},
	)

	if err != nil {
		t.Fatalf("Unexpected error : %e", err)
	}
}

func TestCreate(t *testing.T) {
	store := NewSecretStore()
	conn := newTestConnection(t, store)

	conn.Start()
	defer conn.Stop()

	ctx, cancel := newTestContext()
	defer cancel()

	// fixture
	store.Save(ctx, Secret{
		Name: "already existing",
	})

	t.Run("nominal", func(t *testing.T) {
		client := infrapb.NewSecretsClient(conn.Dial(ctx))
		_, err := client.Create(
			ctx,
			&infrapb.Secret{
				Name: "nominal",
			},
		)

		if err != nil {
			t.Fatalf("Unexpected error : %s", err)
		}

		if contains, _ := store.Contains(ctx, "nominal"); !contains {
			t.Fatalf("Secret not stored")
		}
	})

	t.Run("with an already existing Secret", func(t *testing.T) {
		client := infrapb.NewSecretsClient(conn.Dial(ctx))
		_, err := client.Create(
			ctx,
			&infrapb.Secret{
				Name: "already existing",
			},
		)

		if err == nil {
			t.Fatal("Should not be able to store an already stored secret")
		}
	})

	t.Run("with expiration date", func(t *testing.T) {
		timeIn10s := time.Now().Unix() + 10

		client := infrapb.NewSecretsClient(conn.Dial(ctx))
		_, err := client.Create(
			ctx,
			&infrapb.Secret{
				Name: "valid expiration date",
				Claims: map[string]string{
					"exp": fmt.Sprint(timeIn10s),
				},
			},
		)

		if err != nil {
			t.Fatalf("Unexpected error : %s", err)
		}

		secret, err := store.Fetch(ctx, "valid expiration date")

		if err != nil {
			t.Fatalf("could not fetch Secret")
		}

		if secret.ExpiresAt.Unix() != timeIn10s {
			t.Fatalf("Secret expiration not properly overwritten")
		}
	})

	t.Run("with invalid expiration date", func(t *testing.T) {
		client := infrapb.NewSecretsClient(conn.Dial(ctx))
		_, err := client.Create(
			ctx,
			&infrapb.Secret{
				Name: "valid expiration date",
				Claims: map[string]string{
					"exp": "foo bar baz",
				},
			},
		)

		if err == nil {
			t.Fatal("Expected a invalid argument error, got none")
		}

		if contains, _ := store.Contains(ctx, "foo"); contains {
			t.Fatalf("Secret was stored anyway, shouldn't be the case")
		}
	})
}
