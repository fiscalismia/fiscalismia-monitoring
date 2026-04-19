#!/usr/bin/env bash

readonly BRIGHT_GREEN=$'\033[92m'
readonly CYAN=$'\033[36m'
readonly MAGENTA=$'\033[35m'
readonly RESET=$'\033[0m'

log_msg () {
  printf "${CYAN}===>${MAGENTA} $1${RESET}\n"
}

log_msg "Establishing ssh connection to remote."
ssh demo << 'EOF'
  echo stopping running container
  docker-compose stop fiscalismia-healthcheck || true
  echo launching new container
  docker compose up --no-deps fiscalismia-healthcheck -d
  sleep 5
  echo checking container logs
  podman logs --follow fiscalismia-healthcheck
EOF