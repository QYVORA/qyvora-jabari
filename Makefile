# JABARI Makefile

BINARY ?= jabari
ALIAS  ?= androidsec
PREFIX ?= /usr/local

VERSION ?= dev
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
USER    ?= $(shell id -u -n)

LDFLAGS := -s -w \
	-X github.com/anomalyco/qyvora-jabari/internal/version.Version=$(VERSION) \
	-X github.com/anomalyco/qyvora-jabari/internal/version.Commit=$(COMMIT) \
	-X github.com/anomalyco/qyvora-jabari/internal/version.Date=$(DATE) \
	-X github.com/anomalyco/qyvora-jabari/internal/version.BuildUser=$(USER)

.PHONY: all build test vet fmt check install uninstall clean

all: build

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/jabari
	ln -sf $(BINARY) bin/$(ALIAS)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w cmd internal pkg

check: fmt vet test

install: build
	install -m 0755 bin/$(BINARY) $(PREFIX)/bin/$(BINARY)
	ln -sf $(BINARY) $(PREFIX)/bin/$(ALIAS)

uninstall:
	rm -f $(PREFIX)/bin/$(BINARY) $(PREFIX)/bin/$(ALIAS)

clean:
	rm -rf bin
