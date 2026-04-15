# Fiscalismia Monitoring

Unified Monitoring Endpoint written in Go, concurrently requesting health and status information from all available Fiscalismia Resources to output ASCII formatted data formatted for CLI client queries via e.g. curl from remote devices such as mobile phones, allowing an unified view of system landscape health.

## Setup

```bash
# Install go on fedora
sudo dnf install -y --quiet golang
```

## Running

```bash
cd ~/git/fiscalismia-monitoring/
go mod tidy
go run ./cmd/healthcheck/
```

## Updating


## Testing

```bash
gofmt -s -w .
go vet ./...
go build ./cmd/healthcheck/
./healthcheck
```

## License

MIT License - See [LICENSE](LICENSE) for details.