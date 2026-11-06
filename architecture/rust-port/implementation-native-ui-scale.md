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
preference store has 96.91% line coverage.

## Native text controls

The desktop owner loads `options.yaml` and supplies validated scale options to
the GUI. Tools → Preferences exposes text sizes from 75% to 200%, matching the Go
UI range. Changes save through the application owner before the next frame sees
them. A failed save reports a command notice and retains the previous scale.

Layout, clips, pointer coordinates, and projected dimension labels use the
product of content scale and native backing density. The two inputs remain
separate. The font atlas is rebuilt at that raster size and retained by the
native frame compiler before publication. Returning to 100% restores the default
atlas bytes. Font raster sizes above 256 pixels are refused before allocation;
the default font is 13 pixels.

Floating file, parameter, and dimension dialogs retain valid user positions and
sizes. They fit the available content viewport after a resize or scale change,
including when a collapsed dialog is expanded. The model browser keeps its
300-unit width in wide windows and leaves the view centre available in narrow
windows. These changes keep controls and model picks reachable at 200%.

Native OS mouse and keyboard acceptance passed at backing ratio 1. The run
changed text size from 100% to 200%, created `size = 10 mm` through the parameter
dialog, and authored an anchored XY rectangle with both dimensions driven by
`size`. MCP queries confirmed healthy DOF zero, the two dimensional expressions,
coincident/horizontal/vertical relations, and model extents. It changed size to
20 mm, undid and redid that edit, saved, resized, and closed. A fresh process
loaded the stored 200% preference and reopened the Part. Resetting text to 100%
retained the stored icon factor and unknown option keys and left Part bytes
unchanged. Both desktops closed normally; MCP PNG captures were inspected.

Icon assets and independent icon rendering controls, desktop content-scale
detection, and actual native backing ratio 2 acceptance remain open. This change
does not complete #3850 or the profile/Extrude journey.

The final application gate passed across all 13 crates, including all 389 head
tests. GUI coverage was 98.37%, head coverage 92.64%, and workspace duplication
1.02%. The new scale-owner module had 100% line coverage. Cross-target Clippy
passed for head and GUI on Windows x64 and macOS ARM64. These are compile checks;
the native runtime acceptance above is X11.
