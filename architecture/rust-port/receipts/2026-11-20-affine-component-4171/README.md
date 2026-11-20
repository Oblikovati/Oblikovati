# Exact affine certificate evidence

This receipt records the solver tests and a synthetic release probe for the
explicit affine certificate. Run the probe from the repository root with:

```sh
cargo run --locked --release -p oblikovati-solve --example affine_certificate_probe
```

The certificate assumes the supplied binary64 values are exact binary
rationals. It does not validate a sketch, its parameter sources, or the
rounded result against a sketch tolerance. The probe does not run the GUI or
measure frame rate. Two existing manual solver benchmarks were ignored in the
test suite and are not counted as passes.
