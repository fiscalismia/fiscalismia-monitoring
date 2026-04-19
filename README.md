# Fiscalismia Monitoring

Runs two monitoring services
1: unified HTTP Endpoint written in Go, concurrently requesting health and status information from the entire Fiscalismia System Landscape, returned as ASCII output to curl CLI queries.
2: Grafana Dashboard with Prometheus Metrics, e.g. connection logging for incoming (via LB) and outgoing (via NAT-GW) requests.

### Setup

```bash
# Install go on fedora
sudo dnf install -y --quiet golang
cd ~/git/fiscalismia-monitoring/golang
make tidy
```

### Development

**Compiles automatically via Hot Module Replacement:**

```bash
make air
```

**Compile manually with each change:**

```bash
# runs make clean fmt vet build execute
make server
```

**INFO:** lint before push

```bash
make lint
```

### Updating

- Update go version in `golang/go.mod`
- ??? Update dependencies in `golang/go.mod`
- Update `GOLINT_V` podman version in `golang/Makefile`
- Update `AIR_V` podman version in `golang/Makefile`
- Update `version` in `Lint with golangci-lint` job in pipeline

### Testing

**Prod Build locally**

```bash
podman build \
  -f "Dockerfile" \
  --build-arg VERSION=local \
  --build-arg BUILD_TIME="$(date)" \
  --build-arg COMMIT="$(git rev-parse --short HEAD)" \
  -t health:latest \
  "."
podman run --rm \
  -p 8445:8445 \
  --name fiscalismia-healthcheck \
  -e "ENVIRONMENT=dev" \
  -e "HEALTHCHECK_ADDR=0.0.0.0:8445" \
  health:latest
```

## Prometheus & Grafana

### Setup

```bash
# todo
```

### Updating

## License

MIT License - See [LICENSE](LICENSE) for details.