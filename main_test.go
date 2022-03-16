package main

import (
	"context"
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
		t.Fatalf("Unexpected error : %s", err)
	}
}
