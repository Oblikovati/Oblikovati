# Sealed report-content diagnostic

The experimental sealed CLI built application commit
`c19fe23e563e249556d87126f529ed72ec791289` and ran the pinned scenario
`Inspect a part file before opening it`. It passed one scenario and 12 steps.
The scenario snapshot was `e76cb749ed43c73a91c24854057da7c890041f51`;
verification passed before and after the run. The feature SHA-256 was
`ce0be44dd1e0ded53c12de6a96465ffafca2307db3029351147c6d9606bf4b8b`.

The CLI used the following task-owned paths. `build-source` was exported from
the application commit; the offline vendor tree was made with `cargo vendor
--locked --offline --versioned-dirs`. The runtime was copied from a local runtime
fixture. The command used the installed Rust 1.99.0 toolchain and Bubblewrap.

```sh
python3 -B tools/bdd/sealed_build_cli.py \
  --source /tmp/rust-provenance-3978-StyNHQzl/build-source \
  --vendor /tmp/rust-provenance-3978-StyNHQzl/vendor \
  --toolchain /home/vmiguel/.rustup/toolchains/1.99.0-x86_64-unknown-linux-gnu \
  --config /tmp/rust-provenance-3978-StyNHQzl/config \
  --target /tmp/rust-provenance-3978-StyNHQzl/target \
  --build-logs /tmp/rust-provenance-3978-StyNHQzl/build-logs \
  --bwrap /tmp/rust-provenance-3978-StyNHQzl/bwrap \
  --repository /home/vmiguel/git/oblikovati-workspace/oblikovati-rust \
  --revision c19fe23e563e249556d87126f529ed72ec791289 \
  --runtime /tmp/rust-provenance-3978-StyNHQzl/runtime \
  --checkout /tmp/rust-provenance-3978-StyNHQzl/scenarios \
  --reports /tmp/rust-provenance-3978-StyNHQzl/reports \
  --run-log /tmp/rust-provenance-3978-StyNHQzl/native.log \
  --area application --feature document-opening \
  --scenario 'Inspect a part file before opening it' \
  --registry-archives /home/vmiguel/.cargo/registry/cache/index.crates.io-1949cf8c6b5b557f
```

The same runner SHA-256 appeared in the build and run observations:
`f15d954f5687613dc7fbe924895ca55fb8cc16177fdc4b8acf304e1eceb35a33`.
The registry check covered 280 packages; Git vendor verification was not run.
The in-process report check passed. The final checker added source-root and
malformed-JUnit checks after the process started; a separate process rechecked
these same outputs with the final checker and passed. The 73 focused Python
checks passed, including bad report, source, cleanup, capture, and outline cases.

This is frame-loop diagnostic evidence, not OS-level input acceptance. The
runner hash does not establish how it was built, and a report-content check does
not authenticate its origin. The observation remains `reusable: false` and
runtime coverage remains `unbounded`. Mutable host tools, system and runtime
inputs, Git dependencies, a trusted builder, and authenticated attestations
remain unresolved under #3978; no BDD scenario becomes verified from this run.
