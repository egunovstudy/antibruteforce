.PHONY: run up down test build fmt

run: up

up:
	docker compose up --build

down:
	docker compose down -v

test:
	go test -race ./...

build:
	mkdir -p bin
	go build -o bin/antibf-server ./cmd/server
	go build -o bin/antibf-cli ./cmd/antibf

fmt:
	gofmt -w .
