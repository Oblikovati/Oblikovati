# Native UI content scale

Issue [#3850](https://github.com/Oblikovati/Oblikovati/issues/3850) tracks the
Go font/icon preferences and their native Rust controls. Content size is separate
from framebuffer density. The X11 adapter still uses backing ratio 1.

## Preference foundation

`UiScaleOptions` keeps independent text and icon factors. Finite edits use the
Go limits: text 0.5–2.0 and icons 0.5–2.5. Missing or finite non-positive stored
values use 1.0. Non-finite values are refused with the offending values in the
error; they must never reach GUI geometry.

`ApplicationPreferenceFile` owns sketch and UI groups from Go's `options.yaml`.
The existing `SketchPreferenceFile` name is a type alias. Theme selection remains
in the separate `preferences.yaml` file. Unknown application groups and UI keys
are retained. Each group save reads the current file before atomic replacement,
so sequential owners of different groups retain each other's completed writes.
This does not provide concurrent-process locking.

New values become visible only after a complete atomic save. Read, parse, stage,
write, sync, and commit errors keep the owner's previous values. Application
options are separate from Part persistence and document undo history.

## Verification and remaining work

Four model tests and six UI preference tests passed, together with the existing
seven sketch preference tests. They cover bounds, defaults, independent factors,
invalid/non-finite values, unknown-key retention, interleaved group saves, and
failure without publishing changed values. The full application gate passed
across all 13 crates. The scale model has 100% line coverage and the application
preference store has 96.91%; workspace duplication is 1.01%. No native scale
acceptance is claimed for this preference-only change.

This foundation does not yet load scale into the native GUI or expose its
controls. The remaining work includes the native preference owner, font/spacing
and icon sizing, UI controls, and native mouse/keyboard acceptance at 100% and
200% content scale. Actual backing ratio 2 also needs a platform adapter with
separate logical and framebuffer extents; GUI transform tests alone do not prove
that native behavior.
