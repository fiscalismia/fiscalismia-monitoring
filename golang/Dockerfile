ARG VERSION COMMIT BUILD_TIME

RUN go build -ldflags="-s -w \
  -X 'github.com/fiscalismia/fiscalismia-monitoring/internal/version.Version=${VERSION}' \
  -X 'github.com/fiscalismia/fiscalismia-monitoring/internal/version.Commit=${COMMIT}' \
  -X 'github.com/fiscalismia/fiscalismia-monitoring/internal/version.BuildTime=${BUILD_TIME}'" \
  -trimpath -o /out/healthcheck ./cmd/healthcheck