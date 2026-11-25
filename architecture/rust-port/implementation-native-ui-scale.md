# Native UI content scale

Issue [#3850](https://github.com/Oblikovati/Oblikovati/issues/3850) tracks the
Go font/icon preferences and their native Rust controls. Content size is separate
from framebuffer density.

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

## Windows monitor DPI

The Windows viewport requests per-monitor-v2 DPI awareness before its first native window is created. Native windows report their monitor DPI to the GUI as a physical backing scale; the existing user font and icon preferences remain separate. The main and detached drawing windows rasterize fonts at the effective scale, convert pointer coordinates into logical UI units, and re-create drawing texture resources after a scale change. A DPI-change message applies the suggested window bounds and schedules a new frame. Offscreen views retain their 1× backing scale.

On Windows at 150% display scaling, the previous application process reported DPI-unaware and 96 DPI. The validation build reported per-monitor awareness and 144 DPI on the same display. The 1x, 1.5x and 2x backing-scale GUI hit-region regression passed with the pinned Windows MSVC toolchain. This measurement does not cover movement between monitors or all Windows display configurations.

The Windows full gate was attempted after merging `17ddaee72`. It did not pass: the BDD tooling reported Windows no-follow/config-environment failures, the workspace formatter reported existing differences in files from the pulled branch, and repository checks could not find npm. Workspace doctests completed. These failures are separate from the focused DPI checks and remain unresolved.

## Linux X11 display DPI

Linux native windows use `Xft.dpi` from `RESOURCE_MANAGER` when it contains a
valid value. They otherwise read `Xft/DPI` from the XSETTINGS manager for the
screen. COSMIC at 150% publishes 144 DPI through XSETTINGS. When neither
source supplies a valid value, the backing density defaults to 96 DPI.
[XSETTINGS client and property format](https://specifications.freedesktop.org/xsettings/latest/)

An X resource change, XSETTINGS property change, or settings manager
replacement refreshes the scale and schedules a new frame when the density
changes. The main window and detached drawings share the same backing-scale
path, including font rasterization and coordinate conversion.

The isolated Xvfb regression exercises a 144 DPI XSETTINGS startup, a File
click through X11, and a detached drawing. It checks live XSETTINGS changes,
X resource precedence, fallback after removing the resource, and the default
after the settings manager exits. Both windows are checked at 150%, 200%, and
100%.

## Automatic X11 layout size

The X11 window still reports the configured backing density and renders at
the full drawable extent. With **File → Options → UI → Text size → Automatic**,
the application uses half of the desktop's scale increase above 100% for
layout and font rasterization. A 150% desktop uses 125% UI sizing, and a
200% desktop uses 150%. Explicit text multipliers, including 100%, use the
full desktop density. Automatic is represented by an absent font override
in `options.yaml`. Switching back to Automatic preserves the icon factor.

Main and detached windows share the same automatic sizing calculation.
The 4K arrangement and comparison are recorded in the
[visual receipt](receipts/2026-11-25-x11-4k-ui-scale/README.md).
