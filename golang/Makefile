# Usage: make <target>
# Go convention: Makefiles are thin wrappers around go commands.
# Unlike npm scripts, Go's toolchain is batteries-included, so
# the Makefile mostly just saves you from remembering flags.
BINARY    := healthcheck
CMD       := ./cmd/healthcheck/
SRV_BINARY:= healthcheck-server
SRV_CMD   := ./cmd/server/
IMAGE     := ghcr.io/fiscalismia/fiscalismia-healthcheck
VERSION   := $(shell jq -r '.version' version.json)
COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILDTIME := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
PKG       := github.com/fiscalismia/fiscalismia-monitoring/internal/version
LDFLAGS   := -s -w \
            -X '$(PKG).Version=$(VERSION)' \
            -X '$(PKG).Commit=$(COMMIT)' \
            -X '$(PKG).BuildTime=$(BUILDTIME)'

## phony marks these commands as independent of files on the system .e.g. the healthcheck would otherwise not run if a healthcheck binary file is present
.PHONY: build run lint fmt vet tidy docker clean healthcheck server-build server server-run

##          ___           ___       __        ___  __
##    |__| |__   /\  |     |  |__| /  ` |__| |__  /  ` |__/
##    |  | |___ /~~\ |___  |  |  | \__, |  | |___ \__, |  \
## build: compile the binary for current platform with version metadata stamped in via -ldflags
build:
	go build -ldflags="$(LDFLAGS)" -trimpath -o $(BINARY) $(CMD)

## run: build and run locally (requires certs or use --insecure flag you'd add later)
run: build
	./$(BINARY)

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
	rm -f $(SRV_BINARY)

## execute runs the compiled binary
healthcheck:
	./$(BINARY)

##     __   ___  __        ___  __
##    /__` |__  |__) \  / |__  |__)
##    .__/ |___ |  \  \/  |___ |  \
## server-build: compile the HTTP server binary
server-build:
	go build -ldflags="$(LDFLAGS)" -trimpath -o $(SRV_BINARY) $(SRV_CMD)

## server: build and run the HTTP server locally (loopback-only)
server: server-build
	./$(SRV_BINARY)

## server-run: `go run` the server without producing a binary (fastest dev loop)
server-run:
	go run $(SRV_CMD)
