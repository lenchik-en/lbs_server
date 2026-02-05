#!/bin/bash
set -e

echo "==> Installing system dependencies..."

if ! command -v apt >/dev/null 2>&1; then
  echo "This script supports Debian/Ubuntu only"
  exit 1
fi

sudo apt update

sudo apt install -y \
  ca-certificates \
  curl \
  git \
  postgresql-client

echo "==> Checking Go..."

if ! command -v go >/dev/null 2>&1; then
  echo "Go not found, installing..."

  GO_VERSION=1.23
  ARCH=amd64

  curl -LO https://go.dev/dl/go${GO_VERSION}.linux-${ARCH}.tar.gz
  sudo rm -rf /usr/local/go
  sudo tar -C /usr/local -xzf go${GO_VERSION}.linux-${ARCH}.tar.gz
  rm go${GO_VERSION}.linux-${ARCH}.tar.gz

  echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile
  export PATH=$PATH:/usr/local/go/bin
fi

go version

echo "==> Installing migrate tool..."

if ! command -v migrate >/dev/null 2>&1; then
  curl -L https://github.com/golang-migrate/migrate/releases/latest/download/migrate.linux-amd64.tar.gz \
    | tar xvz
  sudo mv migrate /usr/local/bin/
fi

migrate -version

echo "==> install.sh completed successfully"
