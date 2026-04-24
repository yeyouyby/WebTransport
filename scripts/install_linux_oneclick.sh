#!/usr/bin/env bash
set -euo pipefail

if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  echo "please run as root: bash scripts/install_linux_oneclick.sh"
  exit 1
fi

export DEBIAN_FRONTEND=noninteractive

need_cmd() {
  command -v "$1" >/dev/null 2>&1
}

install_with_apt() {
  apt-get update -y
  apt-get install -y ca-certificates curl jq unzip git golang-go iproute2 procps
}

install_with_dnf() {
  dnf install -y ca-certificates curl jq unzip git golang iproute procps-ng
}

install_with_yum() {
  yum install -y ca-certificates curl jq unzip git golang iproute procps-ng
}

install_with_pacman() {
  pacman -Sy --noconfirm ca-certificates curl jq unzip git go iproute2 procps-ng
}

install_with_zypper() {
  zypper -n install ca-certificates curl jq unzip git go iproute2 procps
}

if need_cmd apt-get; then
  install_with_apt
elif need_cmd dnf; then
  install_with_dnf
elif need_cmd yum; then
  install_with_yum
elif need_cmd pacman; then
  install_with_pacman
elif need_cmd zypper; then
  install_with_zypper
else
  echo "unsupported package manager"
  exit 1
fi

echo "linux one-click install finished"
echo "verify tools:"
echo "  - $(curl --version | head -n 1)"
echo "  - $(jq --version)"
echo "  - $(go version)"
echo "  - $(git --version)"
echo ""
echo "next: run pressure test"
echo "  bash scripts/run_pressure_test.sh --url https://127.0.0.1:8443/fallback --requests 1000 --concurrency 20"
