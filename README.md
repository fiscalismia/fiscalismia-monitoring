# Fiscalismia Monitoring

Unified Monitoring Endpoint written in Go, concurrently requesting health and status information from all available Fiscalismia Resources to output ASCII formatted data formatted for CLI client queries via e.g. curl from remote devices such as mobile phones, allowing an unified view of system landscape health.

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
### Updating

## License

MIT License - See [LICENSE](LICENSE) for details.