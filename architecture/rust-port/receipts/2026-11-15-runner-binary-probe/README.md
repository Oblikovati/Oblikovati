# Native runner identity probe

A clean task worktree at `91b8dbec0` built the native head test executable with
an isolated Cargo target, debug information disabled and incremental builds
disabled. Its SHA-256 was
`e77f1029be6ce339952ca06f4082791c3c2a8b8cc7baacdc11425f1015477c7f`.
Changing CLI-only error text as shown in `obk-scenario-cli-probe.diff` and
rebuilding the same head test target left that executable hash unchanged. With
that change present, the pinned document-opening scenario passed nine steps in
the isolated runtime. The image and native report are retained.

As a negative control, changing the linked GUI confirmation text as shown in
`obk-scenario-gui-probe.diff` and rebuilding changed the executable SHA-256 to
`7f33f3a8bcdbf3f5a50eb51022df6ced52c743ad8abc88d8e3fea38ab7a149c7`.
Both temporary changes were removed. The compressed logs show the build and
run commands' output; the old scenario snapshot was clean before and after.

This is a feasibility probe, not proof that a receipt is reusable. The runtime
and executable provenance are not independently attested, and the scenario
source checkout exposes its Git internals. The normal ledger remains
conservative (#3974).

```sh
CARGO_TARGET_DIR=/tmp/obk-scenario-binary-target \
CARGO_PROFILE_DEV_DEBUG=0 CARGO_PROFILE_TEST_DEBUG=0 CARGO_INCREMENTAL=0 \
cargo test --locked -p oblikovati-head --lib --no-run --message-format=json
```
