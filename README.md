# Backend Hire Challenge

This is the technical challenge for backend engineering candidates.

The it consists in implementing a small gRPC microservice in Go.

## Your mission

In Stories One's infrastructure, we spawn game servers dynamically in a Kubernetes cluster.
These game servers need to be authenticated by the backend in order to be able to interact with the APIs. To achieve this, we need a service that:

    * generates (JWT) tokens and creates the associated Secret,
    * renews Secrets (generate a new JWT) token when they are about to expire.

We assume that those token will be read elswhere in the infrastructure to be passed as environment
variables to the game servers.

Your mission is to implement this microservice as a gRPC service in Go.

## Implementation

It should be built as a Docker container that listens to a specific port which we can connect to with gRPC.
You can use any database backend you like to simulate the Secret store, however it is strongly
encouraged to abstract the database using an interface and provide just an in-memory implementation that can be used to unit-test the service.

You should focus on code quality (do not forget to write tests) and maintainability.
