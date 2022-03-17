package secrets

import (
	"fmt"
	"testing"
	"time"

	"github.com/Taluu/gabsee-test/generated/infrapb"
)

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

func TestUpdate(t *testing.T) {
	store := NewSecretStore()
	conn := newTestConnection(t, store)

	conn.Start()
	defer conn.Stop()

	ctx, cancel := newTestContext()
	defer cancel()

	now := time.Now()

	// fixture
	defaultExpirationDate, _ := time.ParseDuration(defaultExpirationDuration)

	t.Run("nominal update", func(t *testing.T) {
		store.Save(ctx, Secret{
			Name:      "my secret",
			ExpiresAt: time.Now().Add(defaultExpirationDate),
		})

		client := infrapb.NewSecretsClient(conn.Dial(ctx))
		_, err := client.Update(
			ctx,
			&infrapb.Secret{
				Name: "my secret",
				Claims: map[string]string{
					"exp": fmt.Sprint(now.Add(defaultExpirationDate).Unix() + 10), // add 10 seconds for modification sake
				},
			},
		)

		if err != nil {
			t.Fatalf("Unexpected error : %s", err)
		}

		secret, _ := store.Fetch(ctx, "my secret")

		if secret.ExpiresAt.Sub(time.Unix(now.Unix()+10, 0)) != defaultExpirationDate {
			t.Fatalf("Invalid expiration date set ; expected %s, got %s", now.Add(defaultExpirationDate), secret.ExpiresAt)
		}
	})

	t.Run("refresh token (no claims)", func(t *testing.T) {
		expirationDuration, _ := time.ParseDuration("20s")

		store.Save(ctx, Secret{
			Name:      "my secret",
			ExpiresAt: time.Now().Add(expirationDuration),
		})

		client := infrapb.NewSecretsClient(conn.Dial(ctx))
		_, err := client.Update(
			ctx,
			&infrapb.Secret{
				Name: "my secret",
			},
		)

		if err != nil {
			t.Fatalf("Unexpected error : %s", err)
		}

		secret, _ := store.Fetch(ctx, "my secret")

		if secret.ExpiresAt.Sub(time.Unix(now.Unix(), 0)) != defaultExpirationDate {
			t.Fatalf("Invalid expiration date set ; expected %s difference, got %s", defaultExpirationDate, secret.ExpiresAt.Sub(time.Unix(now.Unix()+10, 0)))
		}
	})

	t.Run("with invalid expiration date", func(t *testing.T) {
		store.Save(ctx, Secret{
			Name:      "my secret",
			ExpiresAt: time.Now(),
		})

		client := infrapb.NewSecretsClient(conn.Dial(ctx))
		_, err := client.Update(
			ctx,
			&infrapb.Secret{
				Name: "my secret",
				Claims: map[string]string{
					"exp": "foo bar baz",
				},
			},
		)

		if err == nil {
			t.Fatal("Expected a invalid argument error, got none")
		}
	})

	t.Run("unexisting secret", func(t *testing.T) {
		client := infrapb.NewSecretsClient(conn.Dial(ctx))
		_, err := client.Update(
			ctx,
			&infrapb.Secret{
				Name: "not existing",
			},
		)

		if err == nil {
			t.Fatal("Should not be able to update a Secret that doesn't exist")
		}
	})
}
