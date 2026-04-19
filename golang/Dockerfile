#     __               __
#    |__) |  | | |    |  \
#    |__) \__/ | |___ |__/

# initialize global scope build args by supplying --build-arg flag in podman build
ARG VERSION
ARG COMMIT
ARG BUILD_TIME
ARG HEALTHCHECK_ADDR
ARG ENVIRONMENT

FROM docker.io/library/golang:1.25.9-alpine AS build

WORKDIR /build-dir

# Fetch dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy Source code
COPY . .

RUN ls -hlasR

# run build, forcing static binary without glibc linked to host os
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -ldflags="-s -w \
  -X 'github.com/fiscalismia/fiscalismia-monitoring/internal/version.Version=${VERSION}' \
  -X 'github.com/fiscalismia/fiscalismia-monitoring/internal/version.Commit=${COMMIT}' \
  -X 'github.com/fiscalismia/fiscalismia-monitoring/internal/version.BuildTime=${BUILD_TIME}'" \
  -trimpath -o ./healthcheck ./cmd/healthcheck

#     __   __   __   __        __  ___    __
#    |__) |__) /  \ |  \ |  | /  `  |  | /  \ |\ |
#    |    |  \ \__/ |__/ \__/ \__,  |  | \__/ | \|

# the static distroless images are only about 2MB
# they do not include libraries such as libc
# they do however include ca certs, tzdata, a basic filesystem structure
FROM --platform=linux/amd64 gcr.io/distroless/static-debian13:nonroot

WORKDIR /fiscalismia-monitoring-healthcheck

COPY --from=build /build-dir/targets.yml /build-dir/healthcheck ./

# consume build arguments to expose them in subsequent stages
ARG HEALTHCHECK_ADDR
ARG ENVIRONMENT
# init environment variables /w build arguments, then read in supervisord.conf
ENV HEALTHCHECK_ADDR=$HEALTHCHECK_ADDR
ENV ENVIRONMENT=$ENVIRONMENT

USER 1001:1001

CMD ["./healthcheck"]