#!/usr/bin/env bash
# Run the pincode benchmarks and save the results.
#
#   scripts/bench.sh                 # run, save to bench/results/<timestamp>.txt
#   scripts/bench.sh -count 10       # more runs per benchmark (default 6)
#   scripts/bench.sh -compare FILE   # also compare against an earlier results file (uses benchstat)
#
# Commit a results file as bench/results/baseline.txt to track regressions:
#   scripts/bench.sh -compare bench/results/baseline.txt
set -euo pipefail
cd "$(dirname "$0")/.."

count=6
compare=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    -count)   count="$2"; shift 2 ;;
    -compare) compare="$2"; shift 2 ;;
    *) echo "unknown flag: $1" >&2; exit 2 ;;
  esac
done

mkdir -p bench/results
out="bench/results/$(date +%Y%m%d-%H%M%S).txt"

echo "== tests" >&2
go test -count=1 ./... >&2

echo "== benchmarks (count=$count) -> $out" >&2
{
  echo "# go: $(go version)"
  echo "# commit: $(git rev-parse --short HEAD 2>/dev/null || echo none)"
  echo "# ranges: $(($(wc -l < data/pincode_city_state_ranges.csv) - 1))"
  go test -run '^$' -bench . -benchmem -count "$count" .
} | tee "$out"

if [[ -n "$compare" ]]; then
  if ! command -v benchstat >/dev/null; then
    echo "== installing benchstat" >&2
    go install golang.org/x/perf/cmd/benchstat@latest
    export PATH="$PATH:$(go env GOPATH)/bin"
  fi
  echo "== comparison: $compare vs $out" >&2
  benchstat "$compare" "$out"
fi
