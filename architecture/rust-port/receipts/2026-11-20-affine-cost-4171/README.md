# Dense affine KKT timing probe

This receipt measures the test-only exact-rational elimination prototype from
the Relax Mode worktree. The probe solves equality chains with one preferred
coordinate. It checks the exact result and reports three timings for each size.

From the repository root, copy the archived inputs to a temporary directory,
then build and run them:

```sh
probe_dir=$(mktemp -d)
probe_inputs=architecture/rust-port/receipts/2026-11-20-affine-cost-4171/probe
mkdir -p "$probe_dir/src"
cp "$probe_inputs/Cargo.toml.txt" "$probe_dir/Cargo.toml"
cp "$probe_inputs/Cargo.lock.txt" "$probe_dir/Cargo.lock"
cp "$probe_inputs/src/main.rs.txt" "$probe_dir/src/main.rs"
cargo run --locked --release --manifest-path "$probe_dir/Cargo.toml"
```

The recorded timings are in `result.txt`; `summary.json` records the machine,
dependency versions and input hashes. The process peak memory is not an
application memory comparison. This synthetic probe does not exercise a real
sketch, rendering, or pointer input, and does not establish a frame rate.
