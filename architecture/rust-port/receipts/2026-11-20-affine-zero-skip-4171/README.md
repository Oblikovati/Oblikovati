# Exact affine elimination comparison

This synthetic probe compares the previous dense elimination with a version
that skips exact zero factors and coefficients. It checks 240 deterministic
small systems for equal results, exact feasibility and KKT stationarity.

To reproduce it from the repository root, copy the archived inputs to a
temporary directory and run each mode:

```sh
probe_dir=$(mktemp -d)
probe_inputs=architecture/rust-port/receipts/2026-11-20-affine-zero-skip-4171/probe
mkdir -p "$probe_dir/src"
cp "$probe_inputs/Cargo.toml.txt" "$probe_dir/Cargo.toml"
cp "$probe_inputs/Cargo.lock.txt" "$probe_dir/Cargo.lock"
cp "$probe_inputs/src/main.rs.txt" "$probe_dir/src/main.rs"
cargo build --locked --release --manifest-path "$probe_dir/Cargo.toml"
"$probe_dir/target/release/obk-affine-sparse-4171" corpus
"$probe_dir/target/release/obk-affine-sparse-4171" dense
"$probe_dir/target/release/obk-affine-sparse-4171" sparse
"$probe_dir/target/release/obk-affine-sparse-4171" dense-cases
```

The chain modes use nine samples per size; the denser cases use one sample.
The logs and `summary.json` record the environment and memory readings. The
probe does not derive equations from a sketch and does not test rendering,
pointer input, or application memory. It does not establish a frame rate.
