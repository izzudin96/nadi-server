BINARY := nadi-server
VERSION := $(shell git describe --tags --always 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build build-linux build-full test lint run web clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/$(BINARY)

build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY)-linux-amd64 ./cmd/$(BINARY)

# Build the frontend into internal/webui/dist so the binary embeds it.
web:
	npm --prefix web run build
	rm -rf internal/webui/dist
	cp -r web/dist internal/webui/dist

# Full production build: frontend + binary.
build-full: web build

test:
	go test ./...

lint:
	gofmt -l . && go vet ./...

run:
	go run ./cmd/$(BINARY)

clean:
	rm -rf bin
