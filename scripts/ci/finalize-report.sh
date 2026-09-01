#!/usr/bin/env bash
set -euo pipefail

input_report="$1"
output_report="$2"
stage_metrics="$3"

pr_number="${CI_PR_NUMBER:-}"
merge_sha="${CI_MERGE_SHA:-}"
run_id="${GITHUB_RUN_ID:-}"
event_name="${GITHUB_EVENT_NAME:-unknown}"
sha="${GITHUB_SHA:-}"
repository="${GITHUB_REPOSITORY:-unknown}"
workflow="${GITHUB_WORKFLOW:-gooo-bootstrap-fixed-point}"
artifact_name="${CI_ARTIFACT_NAME:-gooo-bootstrap-report-${run_id}}"

jq \
  --slurpfile stages "$stage_metrics" \
  --arg repository "$repository" \
  --arg workflow "$workflow" \
  --arg event "$event_name" \
  --arg sha "$sha" \
  --arg run_id "$run_id" \
  --arg pr_number "$pr_number" \
  --arg merge_sha "$merge_sha" \
  --arg artifact_name "$artifact_name" \
  '
  .metrics.stages = $stages[0].stages |
  .ci = {
    repository: $repository,
    workflow: $workflow,
    event: $event,
    sha: $sha,
    pull_request_number: (if $pr_number == "" then null else ($pr_number|tonumber) end),
    merge_sha: (if $merge_sha == "" then null else $merge_sha end),
    main_run_id: (if $event == "push" then $run_id else null end),
    job: "verify",
    artifact: {
      name: $artifact_name,
      digest: null,
      decision: "UNKNOWN",
      unknown: {
        stage: "CI",
        step: "READ_ARTIFACT_DIGEST",
        reason: "the digest is emitted by upload-artifact after this report is serialized",
        unknown_class: "POST_UPLOAD_METADATA",
        next_operation: "READ_UPLOAD_ARTIFACT_OUTPUT",
        blocked_by: ["artifact upload"]
      }
    }
  } |
  .authority.remote = {
    repository_writes: 0,
    input_source_writes: 0,
    runtime_commit_merge_tag_release: 0,
    caller_owned_output_writes: 0,
    github_actions_artifact_uploads: 1,
    cross_project_required_gates: 0
  } |
  .authority.operator = {
    repository_create: 0,
    pr_create: 0,
    merge: 0,
    annotated_tag: 0,
    release_create: 0,
    asset_uploads: 0
  } |
  .local_validation_commands = 0 |
  .operational_refuted = false
  ' "$input_report" > "$output_report"
