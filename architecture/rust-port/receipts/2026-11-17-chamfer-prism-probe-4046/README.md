# Chamfer prism probe (#4046)

The pinned Martensite revision is `bef25f47548431c8f128f08f66a1f719bcee4982`.
The test copies a 10-unit cube and uses the kernel's symmetric chamfer builder
on each of its 12 source edges, one at a time, at distance 2. It checks the
original volume, result validity and volume, all seven planar face areas and
loop counts, and coverage of result faces by generated, modified or retained
face history. The expected areas and volume come from the analytic cube and
triangular cut, not tessellation.

This probe does not establish unique edge naming, save/reopen, replay, UI
behavior, general chamfer support, or scenario acceptance. The scenario's
fixture and the native runner provenance remain open. No performance claim is
made.

```sh
cargo test -p oblikovati-martensite --lib prism::fillet_prism_history_probe \
  --locked --offline -- --nocapture
```

Both the existing Fillet probe and the new Chamfer probe passed. Clippy passed
for the Martensite adapter library and tests with warnings denied.
