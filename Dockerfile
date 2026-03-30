FROM golang:1.23 AS builder
WORKDIR /app
COPY go.mod ./
COPY . .
RUN go mod download && \
    CGO_ENABLED=0 GOOS=linux go build -o /bin/antibf-server ./cmd/server && \
    CGO_ENABLED=0 GOOS=linux go build -o /bin/antibf-cli ./cmd/antibf

FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=builder /bin/antibf-server /antibf-server
EXPOSE 8080
ENTRYPOINT ["/antibf-server"]
