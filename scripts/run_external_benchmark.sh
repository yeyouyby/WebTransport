#!/usr/bin/env bash
set -euo pipefail

duration=60
concurrency=1
mode="datagram"
endpoint="${BENCH_ENDPOINT:-https://127.0.0.1:8444/wt}"
outdir="${BENCH_OUTDIR:-ops/external/results/$(date +%Y%m%d-%H%M%S)}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --duration)
      duration="$2"
      shift 2
      ;;
    --concurrency)
      concurrency="$2"
      shift 2
      ;;
    --mode)
      mode="$2"
      shift 2
      ;;
    --endpoint)
      endpoint="$2"
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

mkdir -p "$outdir"

echo "timestamp=$(date -Iseconds)" > "$outdir/benchmark.env"
echo "duration=$duration" >> "$outdir/benchmark.env"
echo "concurrency=$concurrency" >> "$outdir/benchmark.env"
echo "mode=$mode" >> "$outdir/benchmark.env"
echo "endpoint=$endpoint" >> "$outdir/benchmark.env"

echo "run_mode=$mode" | tee "$outdir/summary.log"
echo "note=请接入你的真实压测客户端命令" | tee -a "$outdir/summary.log"

if [[ "$mode" == "datagram" ]]; then
  echo "example_cmd=./bin/bench-client --mode datagram --endpoint $endpoint --seconds $duration --concurrency $concurrency" | tee -a "$outdir/summary.log"
else
  echo "example_cmd=./bin/bench-client --mode stream --endpoint $endpoint --seconds $duration --concurrency $concurrency" | tee -a "$outdir/summary.log"
fi

echo "请将真实压测输出保存到 $outdir/raw.log" | tee -a "$outdir/summary.log"
