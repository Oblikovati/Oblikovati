# Build-tool installer test comparison

The focused installer test failed on both clean base `614ec6f3a` and the
source-discovery fix `a7a0b9ee`. For diagnosis only, each checkout temporarily
added the returned error to the assertion message; that instrumentation was
removed after the runs. Both reports show that API release preflight declined
the locked `a8e651bc` revision before the test's expected SDK-source decline.
This comparison does not establish that the full build-tool suite passes.

```sh
CARGO_PROFILE_DEV_DEBUG=0 CARGO_PROFILE_TEST_DEBUG=0 CARGO_INCREMENTAL=0 \
CARGO_TARGET_DIR=/tmp/obk-3971-compare-target \
  cargo test --offline -p oblikovati-build \
  native_validation::tests::a_changed_or_local_sdk_cannot_execute_its_installer \
  -- --exact --nocapture
```

The compressed logs retain both outputs. The API release dependency remains
tracked in #3965; the source discovery fix is #3971.

After integrating the source-size repair and the walker fix, `make docs-lint`
passed at `f2c7aea0f78b96ded6501701a69d8dba92ce031c`, while an unrelated task
worktree was present. `docs-lint.log` records that partial gate; it is not a
full `make gate` result.

After the rectangle visibility fix, `make docs-lint` passed again at
`180ecccd09c045c1dbd823eefcebbd1ca99c0328`. The output is in
`docs-lint-after-visibility.log`.
