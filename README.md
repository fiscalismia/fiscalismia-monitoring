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

```bash
# runs make clean fmt vet build execute
make server
# lint before push
make lint
```
### Updating

- Update go version in `golang/go.mod`
- ??? Update dependencies in `golang/go.mod`
- Update `GOLINT_V` podman version in `golang/Makefile`
- Update `version` in `Lint with golangci-lint` job in pipeline

### Testing

## Prometheus & Grafana

### Setup

```bash
# todo
```

### Updating

## License

MIT License - See [LICENSE](LICENSE) for details.