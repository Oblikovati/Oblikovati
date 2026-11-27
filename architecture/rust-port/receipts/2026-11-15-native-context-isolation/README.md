# Native test context isolation

On base `22ef655c3`, the two Clear Appearance native tests failed with two
threads: one could not create a GUI context because another test owned it.
After the test-only lease change, both tests passed twice with two threads. A
mixed run of two Clear tests, five Body Properties tests, and eight headless
diagnostic tests passed with eight threads. The lease serializes context-owning
tests, so waiting tests can exceed the test runner's 60-second notice.

The GUI crate's existing test lease is shared with the head unit-test binary.
The lease lives until after the native window and GUI context are dropped.
Headless diagnostic tests use the same lease. Production context ownership is
unchanged. This evidence does not establish that the full head suite or the
full project gate passes; separate integration-test binaries do not share this
unit-test lease.

```sh
CARGO_TARGET_DIR=/tmp/obk-native-context-target \
CARGO_PROFILE_DEV_DEBUG=0 CARGO_PROFILE_TEST_DEBUG=0 CARGO_INCREMENTAL=0 \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib -- \
  application_bdd::diagnostics::tests quick_access_toolbar::clear_appearance \
  application_bdd::body_properties --test-threads=8
```

The compressed logs retain the failing baseline, repeated Clear run, mixed
run, and strict head Clippy output. `cargo fmt --all -- --check` passed.
See #3878.
