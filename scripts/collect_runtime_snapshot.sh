#!/usr/bin/env bash
set -euo pipefail

pid=""
seconds=60
interval=2
outdir="${SNAPSHOT_OUTDIR:-ops/external/results/$(date +%Y%m%d-%H%M%S)}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --pid)
      pid="$2"
      shift 2
      ;;
    --seconds)
      seconds="$2"
      shift 2
      ;;
    --interval)
      interval="$2"
      shift 2
      ;;
    --outdir)
      outdir="$2"
      shift 2
      ;;
    *)
      echo "unknown arg: $1"
      exit 1
      ;;
  esac
done

if [[ -z "$pid" ]]; then
  echo "usage: $0 --pid <process_id> [--seconds 60] [--interval 2] [--outdir dir]"
  exit 1
fi

mkdir -p "$outdir"
outfile="$outdir/runtime-snapshot.csv"

echo "timestamp,cpu_percent,rss_kb,vsz_kb,threads" > "$outfile"

start=$(date +%s)
while true; do
  now=$(date +%s)
  if (( now - start >= seconds )); then
    break
  fi

  stats=$(ps -p "$pid" -o %cpu=,rss=,vsz=,nlwp=)
  ts=$(date -Iseconds)
  echo "$ts,$stats" | tr -s ' ' ',' >> "$outfile"
  sleep "$interval"
done

echo "snapshot saved: $outfile"
