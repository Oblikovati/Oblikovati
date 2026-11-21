# Parallel native Clear test comparison

The two existing native Clear Appearance tests were run with two test threads
on Linux under Xvfb and Lavapipe. On clean base `8aa822fad`, one test failed
because the process-global GUI context already had an owner. The same command
on change `6d87e156a` failed for the same reason in the other test. Each run
passed one test and failed one. Each checkout used its own Cargo target.

The change adds no concurrent native test. Its two native Clear tests passed
when run with one test thread. This comparison does not establish that the
parallel head suite is safe or that the full gate passes. Issue #3878 tracks
GUI test isolation; the parallel head test boundary also needs a repair.

```sh
CARGO_PROFILE_DEV_DEBUG=0 CARGO_PROFILE_TEST_DEBUG=0 CARGO_INCREMENTAL=0 \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  quick_access_toolbar::clear_appearance -- --test-threads=2 --nocapture
```

The compressed logs record the separate target paths and complete outputs.
