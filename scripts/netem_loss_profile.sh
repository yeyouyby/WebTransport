#!/usr/bin/env bash
set -euo pipefail

action="${1:-}"
iface="${2:-eth0}"
loss="${3:-30}"

if [[ -z "$action" ]]; then
  echo "usage: $0 <apply|clear> [iface] [loss_percent]"
  exit 1
fi

if [[ "$action" == "apply" ]]; then
  tc qdisc replace dev "$iface" root netem loss "${loss}%"
  tc -s qdisc show dev "$iface"
  exit 0
fi

if [[ "$action" == "clear" ]]; then
  tc qdisc del dev "$iface" root || true
  tc -s qdisc show dev "$iface"
  exit 0
fi

echo "unknown action: $action"
exit 1
