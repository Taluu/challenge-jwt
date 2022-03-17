package secrets

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Taluu/gabsee-test/generated/infrapb"
	"github.com/golang-jwt/jwt"
)

const testSigningKey = "gisberg"

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

	t.Run("existing secret", func(t *testing.T) {
		// fixture
		store.Save(ctx, NewSecret("foo", defaultTTL))

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
			t.Fatalf("Secret should have been deleted")
		}
	})

	t.Run("not existing secret", func(t *testing.T) {
		client := infrapb.NewSecretsClient(conn.Dial(ctx))
		_, err := client.Delete(
			ctx,
			&infrapb.Secret{
				Name: "bar",
			},
		)

		if err != nil {
			t.Fatalf("Unexpected error : %e", err)
		}
	})
}

func TestCreate(t *testing.T) {
	store := NewSecretStore()
	conn := newTestConnection(t, store)

	conn.Start()
	defer conn.Stop()

	ctx, cancel := newTestContext()
	defer cancel()

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
		// fixture
		store.Save(ctx, NewSecret("already existing", defaultTTL))

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

	t.Run("JWT token is created and stored", func(t *testing.T) {
		client := infrapb.NewSecretsClient(conn.Dial(ctx))
		_, err := client.Create(
			ctx,
			&infrapb.Secret{
				Name: "jwt",
				Claims: map[string]string{
					"Foo": "bar value",
				},
			},
		)

		if err != nil {
			t.Fatalf("Unexpected error : %s", err)
		}

		secret, err := store.Fetch(ctx, "jwt")

		if err != nil {
			t.Fatalf("an error occured while saving the secrt : %s", err)
		}

		token, _ := jwt.Parse(secret.Token, func(token *jwt.Token) (interface{}, error) {
			return []byte(testSigningKey), nil
		})

		if !token.Valid {
			t.Fatalf("A valid token should have been stored")
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

	t.Run("nominal update", func(t *testing.T) {
		store.Save(ctx, NewSecret("my secret", defaultTTL))

		client := infrapb.NewSecretsClient(conn.Dial(ctx))
		_, err := client.Update(
			ctx,
			&infrapb.Secret{
				Name: "my secret",
				Claims: map[string]string{
					"exp": fmt.Sprint(now.Add(defaultTTL).Unix() + 10), // add 10 seconds for modification sake
				},
			},
		)

		if err != nil {
			t.Fatalf("Unexpected error : %s", err)
		}

		secret, _ := store.Fetch(ctx, "my secret")

		if secret.ExpiresAt.Sub(time.Unix(now.Unix()+10, 0)) != defaultTTL {
			t.Fatalf("Invalid expiration date set ; expected %s, got %s", now.Add(defaultTTL), secret.ExpiresAt)
		}
	})

	t.Run("refresh token (no claims)", func(t *testing.T) {
		expirationDuration, _ := time.ParseDuration("20s")
		storedSecret := NewSecret("my secret", defaultTTL)
		storedSecret.ExpiresAt = time.Now().Add(expirationDuration)

		store.Save(ctx, storedSecret)

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

		if secret.ExpiresAt.Sub(time.Unix(now.Unix(), 0)) != defaultTTL {
			t.Fatalf("Invalid expiration date set ; expected %s difference, got %s", defaultTTL, secret.ExpiresAt.Sub(time.Unix(now.Unix()+10, 0)))
		}
	})

	t.Run("with invalid expiration date", func(t *testing.T) {
		store.Save(ctx, NewSecret("my secret", defaultTTL))

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

	t.Run("Claims are overwritten and unspecified claims are kept as is", func(t *testing.T) {
		storedSecret := NewSecret("my secret", defaultTTL)
		storedSecret.Claims = map[string]string{
			"Foo": "should be kept",
			"Bar": "should be overwritten",
		}

		store.Save(ctx, storedSecret)

		client := infrapb.NewSecretsClient(conn.Dial(ctx))
		_, err := client.Update(
			ctx,
			&infrapb.Secret{
				Name: "my secret",
				Claims: map[string]string{
					"Bar": "was overwritten",
					"Baz": "new key !",
				},
			},
		)

		if err != nil {
			t.Fatalf("Unexpected error : %s", err)
		}

		secret, _ := store.Fetch(ctx, "my secret")

		expectedClaims := map[string]string{
			"Foo": "should be kept",
			"Bar": "was overwritten",
			"Baz": "new key !",
		}

		for k, v := range expectedClaims {
			value, ok := secret.Claims[k]

			if !ok {
				t.Fatalf("Expected a %s claim, didn't get it", k)
			}

			if value != v {
				t.Fatalf("expected %s, got %s for %s claim", v, value, k)
			}
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

	t.Run("JWT token is updated", func(t *testing.T) {
		storedSecret := NewSecret("jwt", defaultTTL)
		store.Save(ctx, storedSecret)

		oldToken := storedSecret.Token

		client := infrapb.NewSecretsClient(conn.Dial(ctx))
		_, err := client.Update(
			ctx,
			&infrapb.Secret{
				Name: "jwt",
				Claims: map[string]string{
					"Foo": "bar value",
				},
			},
		)

		if err != nil {
			t.Fatalf("Unexpected error : %s", err)
		}

		secret, err := store.Fetch(ctx, "jwt")

		if err != nil {
			t.Fatalf("an error occured while saving the secrt : %s", err)
		}

		if secret.Token == oldToken {
			t.Fatalf("The token should have been regenerated")
		}
	})
}

func TestRenewExpiredTokens(t *testing.T) {
	type TestCmp struct {
		expiresAt       time.Time
		shouldBeRenewed bool
		token           string
	}

	now := time.Now()
	ctx := context.TODO()
	store := NewSecretStore()
	signingKey := []byte(testSigningKey)

	tests := map[string]TestCmp{
		"almost expired": TestCmp{expiresAt: now.Add(10 * time.Minute), shouldBeRenewed: true},
		"expired":        TestCmp{expiresAt: now.Add(-5 * time.Minute), shouldBeRenewed: true},
		"still alive":    TestCmp{expiresAt: now.Add(5 * time.Hour), shouldBeRenewed: false},
	}

	for k, v := range tests {
		claims := map[string]string{
			"exp": fmt.Sprint(v.expiresAt.Unix()),
		}

		v.token, _ = createToken(k, claims, signingKey)

		store.Save(
			ctx,
			Secret{
				Name:      k,
				ExpiresAt: v.expiresAt,
				Claims:    claims,
				Token:     v.token,
			},
		)

		tests[k] = v
	}

	config := Config{
		SigningKey: []byte("colonel gisberg"),
	}

	service := NewService(store, config)
	service.renewExpiredSecrets(ctx, signingKey, 20*time.Minute, 5*time.Hour)

	secrets, _ := store.List(ctx)

	for _, v := range secrets {
		test := tests[v.Name]
		result := v.ExpiresAt.Equal(test.expiresAt)

		if result == test.shouldBeRenewed {
			t.Errorf("%s :: Expiration not properly updated (got %s == %s (%v), wanted %v)", v.Name, v.ExpiresAt, test.expiresAt, result, test.shouldBeRenewed)
		}

		result = test.token == v.Token

		if result == test.shouldBeRenewed {
			t.Errorf("%s :: JWT not properly updated (got %v, wanted %v)", v.Name, result, test.shouldBeRenewed)
		}
	}
}
