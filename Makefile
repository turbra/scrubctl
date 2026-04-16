SHELL := /usr/bin/env bash

BINARY := scrubctl
BIN_DIR := $(CURDIR)/bin
VERSION ?= $(shell ./hack/version.sh)
COMMIT ?= $(shell git rev-parse --short HEAD)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)

.PHONY: build install test release

build:
	mkdir -p "$(BIN_DIR)"
	go build -ldflags '$(LDFLAGS)' -o "$(BIN_DIR)/$(BINARY)" ./cmd/$(BINARY)

install:
	go install -ldflags '$(LDFLAGS)' ./cmd/$(BINARY)

test:
	go test ./...

release:
	./hack/release.sh
