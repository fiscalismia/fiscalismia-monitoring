#!/usr/bin/env bash

CURRENT_DIR=$(pwd | sed 's:.*/::')
if [ "$CURRENT_DIR" != "scripts" ]
then
  log_msg "please change directory to scripts folder and execute the shell script again."
  exit 1
fi

readonly BRIGHT_GREEN=$'\033[92m'
readonly CYAN=$'\033[36m'
readonly MAGENTA=$'\033[35m'
readonly RESET=$'\033[0m'

log_msg () {
  printf "${CYAN}===>${MAGENTA} $1${RESET}\n"
}

cd ../golang/

log_msg "formatting go files"
make fmt

log_msg "vetting go files"
make vet

log_msg "building go binary"
make clean
make build

log_msg "creating remote project structure"
ssh loadbalancer mkdir -p /usr/local/etc/go

log_msg "copying binary and target config to remote"
scp ./healthcheck ./targets.yml root@loadbalancer:/usr/local/etc/go/

log_msg "${BRIGHT_GREEN}EXECUTING BINARY ON REMOTE"
ssh loadbalancer << 'EOF'
cd /usr/local/etc/go/
./healthcheck
exit 0
EOF