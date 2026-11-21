# Document Saving native baseline

All 23 pinned scenarios and 218 steps passed on Linux X11 under
Xvfb with Openbox and Vulkan Lavapipe. The application commit was
`e93259292f9a15303d4d63c2c1f3412de51ea2c0`, with API v0.165.0 and Martensite v0.13.1. The
verified source snapshot `b576445b98cffd686dcc05aa8c017de2a591c8c8` remained clean. The pinned
feature SHA-256 was `ee58ca1d01af2e1db700be89fa237cee19f7d42c477fbe6c8f5f62d07683cbd2`.

The [Cucumber report](cucumber.json) covers first save, cancellation,
save and reopen, preview source and capture, folder navigation, file-list
modes, replacement prompts, and the shaded first-save preview.
It records the exact steps for each baseline.

The current feature SHA-256 was `d7e1c5d95bed2449e0086e0dd7a3ec4bbaf8929725df7a8f2af53a620efdfcd3`. Nine scenarios
now name fixture inputs from the source folder, and a new scenario starts
with a Simplify-generated Part from an assembly. The current source needs
its fixture references updated to use the Rust native Part files. The
assembly-dependent case remains deferred.

The [native fixture pack](../../../../test-utilities/fixtures/README.md#document-saving)
was tested separately at `16f6fa110219f0f98c8a03d07a3a26d47a29e4ec`. All three native parts
reopened through the desktop. The block has one body with six faces,
20,000 mm³ volume, and 4,600 mm² area. The inertia checks confirm the
40 × 25 × 20 mm dimensions. The test declined replacing the saved part
and checked both files byte for byte and the unsaved tab state.

Run the pinned suite with `OBK_BDD_FEATURE=document-saving` in Xvfb
with Openbox and the Lavapipe ICD. The test command is:

```sh
cargo test --locked --offline -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture --test-threads=1
```

The [Go persistence tests](../2026-11-14-go-save-open-baseline/README.md)
and [Go dialog tests](../2026-11-15-go-save-preview/README.md) provide
the reference premises. Strict verification remains open under
[#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
