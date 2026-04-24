#!/usr/bin/env bash
set -euo pipefail

mode="fallback"
url="${PRESSURE_URL:-https://127.0.0.1:8443/fallback}"
requests="${PRESSURE_REQUESTS:-1000}"
concurrency="${PRESSURE_CONCURRENCY:-20}"
range_size="${PRESSURE_RANGE_SIZE:-262144}"
timeout_sec="${PRESSURE_TIMEOUT_SEC:-8}"
bench_client="${PRESSURE_BENCH_CLIENT:-./bin/bench-client}"
endpoint="${BENCH_ENDPOINT:-https://127.0.0.1:8444/wt}"
duration="${BENCH_DURATION:-60}"
outdir="${PRESSURE_OUTDIR:-ops/external/results/$(date +%Y%m%d-%H%M%S)}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode)
      mode="$2"
      shift 2
      ;;
    --url)
      url="$2"
      shift 2
      ;;
    --requests)
      requests="$2"
      shift 2
      ;;
    --concurrency)
      concurrency="$2"
      shift 2
      ;;
    --range-size)
      range_size="$2"
      shift 2
      ;;
    --timeout)
      timeout_sec="$2"
      shift 2
      ;;
    --bench-client)
      bench_client="$2"
      shift 2
      ;;
    --endpoint)
      endpoint="$2"
      shift 2
      ;;
    --duration)
      duration="$2"
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

if [[ "$mode" == "bench" ]]; then
  if [[ ! -x "$bench_client" ]]; then
    echo "bench client not found or not executable: $bench_client"
    exit 1
  fi
  "$bench_client" \
    --mode datagram \
    --endpoint "$endpoint" \
    --seconds "$duration" \
    --concurrency "$concurrency" \
    | tee "$outdir/bench.log"
  echo "bench mode completed: $outdir/bench.log"
  exit 0
fi

status_file="$outdir/fallback-http-status.log"
meta_file="$outdir/fallback-summary.env"

echo "timestamp=$(date -Iseconds)" > "$meta_file"
echo "mode=$mode" >> "$meta_file"
echo "url=$url" >> "$meta_file"
echo "requests=$requests" >> "$meta_file"
echo "concurrency=$concurrency" >> "$meta_file"
echo "range_size=$range_size" >> "$meta_file"

export PRESSURE_URL_RUNTIME="$url"
export PRESSURE_RANGE_SIZE_RUNTIME="$range_size"
export PRESSURE_TIMEOUT_RUNTIME="$timeout_sec"

start_epoch_ms=$(date +%s%3N)

seq 1 "$requests" | xargs -P "$concurrency" -I{} bash -c '
  i="$1"
  size="${PRESSURE_RANGE_SIZE_RUNTIME}"
  offset=$(( (i - 1) * size ))
  end=$(( offset + size - 1 ))
  code=$(curl -k -sS -o /dev/null -w "%{http_code}" \
    --connect-timeout "${PRESSURE_TIMEOUT_RUNTIME}" \
    --max-time "${PRESSURE_TIMEOUT_RUNTIME}" \
    -H "Range: bytes=${offset}-${end}" \
    "${PRESSURE_URL_RUNTIME}" || true)
  if [[ -z "$code" ]]; then
    code="000"
  fi
  printf "%s\n" "$code"
' _ {} > "$status_file"

end_epoch_ms=$(date +%s%3N)
elapsed_ms=$(( end_epoch_ms - start_epoch_ms ))
if (( elapsed_ms <= 0 )); then
  elapsed_ms=1
fi

ok_count=$(grep -Ec '^(200|206)$' "$status_file" || true)
total_count=$(wc -l < "$status_file")
fail_count=$(( total_count - ok_count ))
total_bytes=$(( total_count * range_size ))
throughput_mib=$(awk -v b="$total_bytes" -v ms="$elapsed_ms" 'BEGIN { printf "%.2f", (b * 1000.0 / ms) / (1024.0 * 1024.0) }')

echo "elapsed_ms=$elapsed_ms" >> "$meta_file"
echo "ok_count=$ok_count" >> "$meta_file"
echo "fail_count=$fail_count" >> "$meta_file"
echo "throughput_mib_per_sec=$throughput_mib" >> "$meta_file"

echo "pressure test completed"
echo "  status_file: $status_file"
echo "  summary:     $meta_file"
echo "  ok/fail:     $ok_count/$fail_count"
echo "  throughput:  ${throughput_mib} MiB/s"
