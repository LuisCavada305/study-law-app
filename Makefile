SHELL := /bin/bash

.PHONY: dev-api dev-web build run test windows

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
	cd frontend && npm run build  # o go:embed de frontend/ precisa do dist
	go vet ./...
	go test ./...
	cd frontend && npm test

# App desktop para Windows (Wails): um .exe só, com o frontend embutido.
# Compila daqui mesmo, sem cgo; no Windows usa o WebView2 que já vem no sistema.
windows:
	cd frontend && npm ci && npm run build
	# ícone (ID 3, o que o Wails procura), manifesto de DPI e versão
	go run github.com/tc-hib/go-winres@v0.3.3 make --in build/winres.json \
		--out cmd/leiseca-desktop/rsrc --arch amd64
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -tags desktop,production \
		-ldflags "-s -w -H windowsgui" -o bin/LeiSeca.exe ./cmd/leiseca-desktop
