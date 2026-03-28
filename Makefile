VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS = -ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o bin/restless ./cmd/restless

test:
	go test ./... -count=1

vet:
	go vet ./...

lint:
	staticcheck ./...

run:
	go run $(LDFLAGS) ./cmd/restless

install:
	go install $(LDFLAGS) ./cmd/restless

clean:
	rm -rf bin/ .restless/

.PHONY: build test vet lint run install clean
