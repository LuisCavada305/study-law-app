SHELL := /bin/bash

.PHONY: dev-api dev-web build run test

# Desenvolvimento: rode os dois em terminais separados e abra http://localhost:5173
dev-api:
	go run ./cmd/leiseca-server

dev-web:
	cd frontend && npm run dev

# Produção: um binário só, que serve a API e o frontend em http://127.0.0.1:8080
build:
	cd frontend && npm ci && npm run build
	go build -o bin/leiseca-server ./cmd/leiseca-server
	go build -o bin/leiseca-parse ./cmd/leiseca-parse

run: build
	./bin/leiseca-server

test:
	go vet ./...
	go test ./...
	cd frontend && npm test
