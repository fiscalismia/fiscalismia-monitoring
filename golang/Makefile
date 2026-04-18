# Usage: make <target>
# Go convention: Makefiles are thin wrappers around go commands.
# Unlike npm scripts, Go's toolchain is batteries-included, so
# the Makefile mostly just saves you from remembering flags.
BINARY    := healthcheck
CMD       := ./cmd/healthcheck/
IMAGE     := ghcr.io/fiscalismia/fiscalismia-healthcheck
VERSION   := $(shell jq -r '.version' version.json)
COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILDTIME := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
PKG       := github.com/fiscalismia/fiscalismia-monitoring/internal/version
LDFLAGS   := -s -w \
             -X '$(PKG).Version=$(VERSION)' \
             -X '$(PKG).Commit=$(COMMIT)' \
             -X '$(PKG).BuildTime=$(BUILDTIME)'

.PHONY: build run test lint fmt tidy docker clean

## build: compile the binary for current platform with version metadata stamped in via -ldflags
build:
	go build -ldflags="$(LDFLAGS)" -trimpath -o $(BINARY) $(CMD)

## run: build and run locally (requires certs or use --insecure flag you'd add later)
run: build
	./$(BINARY) --config=./targets.yml

## test: run all tests with race detector enabled
## The race detector instruments memory accesses at compile time to catch
## concurrent access bugs. Always enable it in CI — the ~2x slowdown is
## worth catching data races that would be near-impossible to debug in prod.
test:
	go test -race -count=1 -timeout=30s ./...

## lint: run golangci-lint with project config
lint:
	golangci-lint run

## fmt: format all Go files (gofmt is the universal standard, never deviate)
fmt:
	gofmt -s -w .

## vet is a basic golang convention sanity check
vet:
	go vet ./...
## tidy: clean up go.mod and go.sum (run after adding/removing imports)
tidy:
	go mod tidy

## docker: build OCI image with podman/buildah
docker:
	podman build --rm -t $(IMAGE):$(VERSION) -t $(IMAGE):latest .

## clean: remove build artifacts
clean:
	rm -f $(BINARY)

## execute runs the compiled binary
execute:
	./$(BINARY)
