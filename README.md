# Fiscalismia Monitoring

Runs two monitoring services
1: unified HTTP Endpoint written in Go, concurrently requesting health and status information from the entire Fiscalismia System Landscape, returned as ASCII output to curl CLI queries.
2: Grafana Dashboard with Prometheus Metrics, e.g. connection logging for incoming (via LB) and outgoing (via NAT-GW) requests.


## Golang Healthcheck
### Setup

```bash
# Install go on fedora
sudo dnf install -y --quiet golang
cd ~/git/fiscalismia-monitoring/golang
go mod tidy
```

### Running

**Naively**
```bash
go run ./cmd/healthcheck/
```

**Properly**
```bash
gofmt -s -w . && go vet ./...
go build ./cmd/healthcheck/
./healthcheck
```

### Updating
### Testing

## Prometheus & Grafana

### Setup

```bash
# todo
```

### Updating

## License

MIT License - See [LICENSE](LICENSE) for details.