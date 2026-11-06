# Sketch visibility

Date: 2026-11-06. Tracking: [#3826](https://github.com/Oblikovati/Oblikovati/issues/3826).

## Stored and API state

Visibility is saved metadata on each placed sketch. A hidden sketch uses recipe
version 5 with an explicit override. Visible sketches retain the previous recipe
encoding. Versions 1 through 4 default to visible and refuse visibility overrides.
Geometry, stable identities, verification, and model references stay intact.

The existing Apache API `sketch.setProperty` method accepts the `visible` property
and the same boolean spellings as Go `strconv.ParseBool`. Unsupported properties,
invalid indices, invalid boolean values, and whitespace produce named errors.
The normal `SketchInfo` reply is prepared before adoption. A reply inspection
failure rolls back the command. Numerical sketch health does not prevent this
metadata edit when inspection inputs are available. Unavailable inspection inputs
retain the existing query refusal contract.

The command does not solve geometry. A changed flag participates in normal undo,
redo, dirty state, save, reopen, and save-copy. An unchanged flag does not advance
history or generation, including within a caller-owned transaction.

## Desktop path

The model browser provides the Go-compatible checked `Visibility` menu item. The
immutable row includes the saved flag even when its inspection has a local error.
A native action checks active document and stable sketch identity, then calls the
same typed API. Wrappers forward the operation to the document owner.

Hidden sketches leave the scene before work-plane resolution. They also leave
viewport hit testing and dimension overlays. Their model geometry and dimensions
remain available to feature consumers and API inspection. A committed change
retires stale generation-bound tools and picks through the existing shared
context checks. No-op and rejected changes preserve the active tool.

## Verification

Tests cover versioned persistence, legacy encodings, geometry and identity
retention, typed API reply rollback, history, save/reopen and save-copy, browser
mouse clicks, scene filtering, picking, labels, and live tool lifetime. The native
acceptance path uses X11 mouse and keyboard input for mutations and MCPBridge
queries and captures for observation.

The full local gate passed all 13 crates, including all 337 head tests. Line
coverage is 91.96% to 99.60%, duplicated lines are 0.97%, and Markdown lint has
zero issues. Windows x64 GNU and macOS ARM64 host/GUI Clippy checks also passed.
These compile checks do not claim native runtime coverage on those systems.

The live X11 path created and edited a parameter-bound dimension, hid its sketch
through the browser menu, verified that hidden edit clicks select nothing, tested
undo/redo, and saved the hidden sketch. MCP captures showed the points and label
leaving the viewport while the browser row and dimension remained in the model.
A fresh process reopened the hidden sketch, retained its expression and label
anchor, and restored its points and label through the browser Visibility menu.

The complete cube and named-face acceptance remains a separate task. This change
does not add a kernel profile or Extrude implementation.
