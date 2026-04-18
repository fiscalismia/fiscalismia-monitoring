#!/usr/bin/env bash

cd ~/git/fiscalismia-monitoring
gofmt -s -w .
go vet ./...
go build ./cmd/healthcheck/
ssh loadbalancer mkdir -p /usr/local/etc/go
scp ./healthcheck ./targets.yml root@loadbalancer:/usr/local/etc/go/

ssh loadbalancer << 'EOF'
cd /usr/local/etc/go/
./healthcheck
exit 0
EOF