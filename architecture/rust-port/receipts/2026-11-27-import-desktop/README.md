# Import desktop receipt (2026-11-27)

## Summary

Verified all 12 Part **Import** desktop BDD scenarios (`import-01` through
`import-12`, 66/66 steps) on native Wayland (`WAYLAND_DISPLAY=wayland-1`), plus
unit, persistence, Martensite STEP/STL exchange, GUI dockable-window guard,
documentation, formatting, and workspace clippy gates.

## Native Wayland BDD

Command:

```bash
WAYLAND_DISPLAY=wayland-1 \
  OBK_BDD_ROOT=/mnt/vininas-projects/MCAD-Scenarios \
  OBK_BDD_AREA=solid-model OBK_BDD_FEATURE=import \
  cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```

Result: 12 scenarios passed, 0 failed, 0 skipped (66/66 steps). See
[wayland-native.txt](wayland-native.txt).

## Additional checks

- `cargo test --locked -p oblikovati-model --lib --test work_planes` (255 passed)
- `cargo test --locked -p oblikovati-persistence --test package_work_planes` (10 passed)
- `cargo test --locked -p oblikovati-martensite --test import` (1 passed)
- `cargo test --locked -p oblikovati-automation --lib` (77 passed)
- `cargo test --locked -p oblikovati-gui --lib --test dockable_window_guard` (858 passed)
- `cargo fmt --all -- --check` (0 diffs)
- `cargo clippy --locked --workspace --all-targets -- -D warnings` (0 warnings)
- `cargo run --locked -p oblikovati-build -- docs` (0 issues)
