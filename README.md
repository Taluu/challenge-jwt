# Backend Hire Challenge

## Usage
To build the service, just do a

```shell
make image
```

You can configure the service with the following env vars :
- `SECRETS_TLL`: The duration secrets are living, By default, it's `24h`, and the format is a string that go's `time.Duration` can parse.
- `SECRETS_NEAR_TTL`: The duration secrets are considered "nearly expired". By default, it's `1h`, and the format is a string that go's `time.Duration` can parse.
- `SECRETS_TICKS`: Determine the time period the servie should check and renew (nearly) expired secrets. By default, it's `1s`, and the format is a string that go's `time.Duration` can parse.
- `SECRETS_JWT_SIGNING_KEY`: The signing key to use when encoding / decoding the stored jwt token. By default, it's empty, but I cannot stress enough that if you want a bit of security, you should give it a value.

Then once you're set, you can do the following :

```shell
docker run -p 50051:50051 dev/secrets
```

You can change the published port, add the env variable to configure the service as you see fit.

-----

## Original subject

This is the technical challenge for backend engineering candidates.

It consists in implementing a small gRPC microservice in Go.

### Disclaimer

All materials you create for this test belong to you. This test is purely for evaluation purposes, we won’t use anything you share with us without your explicit permission.

### Your mission

In Stories One's infrastructure, we spawn game servers dynamically in a Kubernetes cluster.
These game servers need to be authenticated by the backend in order to be able to interact with the APIs.

To achieve this, game servers are passed an auth *Token* through an environment variable upon startup. This environment variable is dynamically read by Kubernetes every time the game server needs to start: its value is stored as a Secret.

Tokens are [JWT](https://jwt.io/), which can be seen as a cryptographically signed map of *claims*.
A token is only valid for a limited period in time, determined by its `"exp"` claim that is his
expiration date, represented as a Unix timestamp (= number of seconds since Jan 1st 1970 00:00 UTC).

Since the generated token is only valid for a limited time, we need to renew the token in every secret we manage, when they are about to expire.

This is the job of the *Secrets* service you're about to write:

* methods that allow to generate (JWT) tokens and create or update the associated Secret,
* a background task that renews Secrets (generate a new JWT token with an updated `exp` claim) when they are about to expire.

Your mission is to implement this microservice as a gRPC service. This task is estimated to 4-6 hours for somebody who's already using Go and you shouldn't spend more than this.

**It is okay if you don't finish**, especially if you're new to Go or gRPC: your code is the base for a discussion.

### Implementation

The `.proto` file that defines the gRPC service is provided: [infra.proto](./infra.proto).

Your server should be built as a Docker container that listens to a specific port which we can connect to with gRPC.

You can use any database backend you like to simulate the Secret store, however it is strongly
encouraged to abstract the database using an interface and provide just an in-memory implementation that can be used to unit-test the service.

You should focus on code quality (do not forget to write tests) and maintainability.

### Delivery

Your solution should be stored in a public git repository and include:
* A Dockerfile that allows to build the service's container.
* Usage instructions in a README file.
* [Optional but appreciated] A Makefile with at least `make`, `make image` and `make tests` targets.
