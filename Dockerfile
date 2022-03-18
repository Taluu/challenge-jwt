FROM golang:1.18
WORKDIR /go/src/github.com/taluu/challenge-jwt/
COPY . ./
RUN mkdir -p bin && \
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/server ./pkg/

FROM alpine:latest
COPY --from=0 /go/src/github.com/taluu/challenge-jwt/bin/server /bin/server
EXPOSE 50051
ENTRYPOINT ["/bin/server"]
