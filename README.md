# Gooo bootstrap fixed point

This repository is an executable proof boundary for moving Gooo toward a
self-improving programming language. The semantic source is
[`examples/self-description.gooo`](examples/self-description.gooo). It declares
the verifier and generator activities, the `G0 → G1 → G2` stages, the
`REFUTED > UNKNOWN > CLOSED` resolution order, the six-field UNKNOWN tuple, and
the exact fixed-point components.

Go is deliberately limited to parsing the declaration, materializing a
canonical semantic IR and provenance graph, generating Go, executing the
generated program, and evaluating evidence. It does not write the input source
or repository. All runtime output is written to the caller-owned output
directory.

## Executable proof

The CI conformance job runs this path:

`.gooo input → semantic IR → provenance graph → generated Go → executed behavior → human-readable report`

G1 and G2 are accepted as `FIXED_POINT_OBSERVED/CLOSED` only when all four
canonical components match exactly as canonical bytes:

1. canonical semantic IR;
2. generated Go behavior (the generated Go source and its behavior contract);
3. generated program output;
4. provenance graph.

A matching digest is evidence displayed in the receipt, never the sole proof.
The bootstrap seed compiler is a separate trust boundary. Its trust remains
`UNKNOWN` until an independent bootstrap proof is supplied, while the observed
G1/G2 regeneration match is reported separately as `CLOSED`.

## Fixed denominator

[`contracts/denominator-v1.json`](contracts/denominator-v1.json) fixes nine
canonical cases: three NORMAL, three UNKNOWN, and three REFUTED. The corpus
includes malformed input and an unknown top-level decision, both of which fail
closed as `REFUTED`. Every UNKNOWN carries `stage`, `step`, `reason`,
`unknown_class`, `next_operation`, and `blocked_by`.

The overall corpus decision is the precedence reduction of the observed case
decisions. Counts are exact counts, not scores, percentages, or weighted sums.
Missing utility evidence is `null + UNKNOWN`; this repository does not claim
external-user utility or improvement without external evidence and an exact
same-job before/after integer pair across scenario, source, contract, fixture,
toolchain, and runner.

## CI and authority

GitHub Actions uses the pinned Go 1.27 toolchain and records compile, build,
test, conformance, and integration `wall_ms` and `peak_rss_kib`. The evidence
artifact also records test totals, selection, execution, reuse, failures,
UNKNOWN cases, Go/Gooo file counts and physical lines, descendant directories,
regular files, generated artifact counts and bytes, and local/remote authority
counts. The root README is explicitly excluded from the source inventory.

The runtime has zero repository, source, commit, merge, tag, and release
authority. CI has `cross_project_required_gates=0` and uses `github.token` only
through standard Actions permissions for evidence upload. Repository creation,
PR creation, merge, annotated tag, release, and release-asset upload are
operator actions recorded separately from runtime authority.

## Reproducible commands

The commands below are the CI entry points. Their outputs must be directed to a
caller-owned directory:

```text
gooo-bootstrap conformance --root . --input examples/self-description.gooo --output <output>
gooo-bootstrap integration --input examples/self-description.gooo --output <output>
```

Local validation is intentionally not part of the operator workflow for this
bootstrap transaction; the authoritative validation is the GitHub Actions
workflow attached to the PR and the merged `main` commit.
