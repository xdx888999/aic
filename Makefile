APP_NAME := aic
BUILD_DIR := bin
VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w \
	-X github.com/xdx888999/aic/internal/version.Version=$(VERSION) \
	-X github.com/xdx888999/aic/internal/version.Commit=$(COMMIT) \
	-X github.com/xdx888999/aic/internal/version.BuildTime=$(BUILD_TIME)

.PHONY: build test run clean release-dry-run

build:
	mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) .

test:
	mkdir -p .gocache
	GOCACHE=$(PWD)/.gocache go test ./...

run:
	go run .

clean:
	rm -rf $(BUILD_DIR)

release-dry-run:
	goreleaser release --snapshot --clean
