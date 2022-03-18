DOCKER ?= docker
REGISTRY ?= dev
TAG ?= latest

.PHONY: all tests server image

all: image

tests:
	go test ./...

local:
	@mkdir -p bin
	go build -o bin/server ./pkg/

generate:
	go generate ./...

image: generate
	$(DOCKER) build -t $(REGISTRY)/secrets:$(TAG) -f ./Dockerfile .
