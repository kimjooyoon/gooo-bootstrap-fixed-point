#!/usr/bin/env bash
set -euo pipefail

repo_root="${GITHUB_WORKSPACE:-$(pwd)}"
ci_output="${CI_OUTPUT:-$repo_root/ci-output}"
work_root="${RUNNER_TEMP:-$repo_root/.ci-temp}/gooo-bootstrap-fixed-point"
binary="$work_root/gooo-bootstrap"
conformance_output="$work_root/conformance"
integration_output="$work_root/integration"
metrics_output="$work_root/stage-metrics.json"

mkdir -p "$work_root" "$ci_output"

measure_stage() {
  local stage="$1"
  shift
  local time_file="$work_root/${stage}.rss"
  local started ended wall rss
  started="$(date +%s%3N)"
  /usr/bin/time -f '%M' -o "$time_file" "$@"
  ended="$(date +%s%3N)"
  wall=$((ended - started))
  rss="$(tr -d '[:space:]' < "$time_file")"
  printf '{"stage":"%s","wall_ms":%s,"peak_rss_kib":%s}\n' "$stage" "$wall" "$rss" > "$work_root/${stage}.json"
}

measure_stage compile go build -o "$binary" ./cmd/gooo-bootstrap
measure_stage build go build ./...
measure_stage test go test ./...
measure_stage conformance "$binary" conformance --root "$repo_root" --input "$repo_root/examples/self-description.gooo" --output "$conformance_output"
measure_stage integration "$binary" integration --input "$repo_root/examples/self-description.gooo" --output "$integration_output"

jq -n \
  --slurpfile compile "$work_root/compile.json" \
  --slurpfile build "$work_root/build.json" \
  --slurpfile test "$work_root/test.json" \
  --slurpfile conformance "$work_root/conformance.json" \
  --slurpfile integration "$work_root/integration.json" \
  '{stages:[$compile[0],$build[0],$test[0],$conformance[0],$integration[0]]}' > "$metrics_output"

cp -R "$conformance_output" "$ci_output/conformance"
cp -R "$integration_output" "$ci_output/integration"
cp "$metrics_output" "$ci_output/stage-metrics.json"
cp "$repo_root/contracts/denominator-v1.json" "$ci_output/denominator-v1.json"
cp "$repo_root/examples/self-description.gooo" "$ci_output/self-description.gooo"
