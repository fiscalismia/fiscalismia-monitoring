GOLINT_V  := v2.11.4 # https://github.com/golangci/golangci-lint
AIR_V     := v1.65.1 # https://github.com/air-verse/air
BINARY    := healthcheck
CMD       := ./cmd/healthcheck/
IMAGE     := ghcr.io/fiscalismia/fiscalismia-healthcheck
VERSION   := $(shell jq -r '.version' version.json)
COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILDTIME := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
PROJECT   := github.com/fiscalismia/fiscalismia-monitoring
PKG       := github.com/fiscalismia/fiscalismia-monitoring/internal/version
LDFLAGS   := -s -w \
            -X '$(PKG).Version=$(VERSION)' \
            -X '$(PKG).Commit=$(COMMIT)' \
            -X '$(PKG).BuildTime=$(BUILDTIME)'

## phony marks these commands as independent of files on the system .e.g. the healthcheck would otherwise not run if a healthcheck binary file is present
.PHONY: tidy run air lint fmt vet docker clean build execute server

## tidy: clean up go.mod and go.sum (run after adding/removing imports)
tidy:
	go mod tidy

## naively run the main directly
run:
	go run $(CMD)

## air: hot module reload during development, overwrite HEALTHCHECK_ADDR to not use loopback for container
air:
	podman run -it --rm \
		-w $(shell pwd) \
		-v $(shell pwd):$(shell pwd) \
		-p 8445:8445 \
		-e "HEALTHCHECK_ADDR=0.0.0.0:8445" \
    -e "ENVIRONMENT=air" \
		docker.io/cosmtrek/air:${AIR_V} \
		-c "$(shell pwd)/.air.toml"

## lint: run golangci-lint with project config
lint:
	podman run -t --rm -v $(shell pwd):/app:z -w /app docker.io/golangci/golangci-lint:${GOLINT_V} golangci-lint run

## fmt: format all Go files (gofmt is the universal standard, never deviate)
fmt:
	gofmt -s -w .

## vet is a basic golang convention sanity check
vet:
	go vet ./...

## docker: build OCI image with podman/buildah
docker:
	podman build --rm -t $(IMAGE):$(VERSION) -t $(IMAGE):latest .

## clean: remove build artifacts
clean:
	rm -f $(BINARY)

## build: compile the binary for current platform with version metadata stamped in via -ldflags
build:
	go build -ldflags="$(LDFLAGS)" -trimpath -o $(BINARY) $(CMD)

## execute: execute the compiled binary
execute:
	./$(BINARY)

## server: full localhost build pipeline integrating all make commands
server: clean fmt vet build execute