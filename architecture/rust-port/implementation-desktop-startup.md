# Native document startup — partial R1V

The Linux head accepts native document operands and opens them through the
canonical typed API client before its first frame. The existing file owner
restores recipes, selects the active document, and supplies evaluated geometry
to the shared scene loop. Startup does not add an exchange reader or a second
model owner.

```sh
cargo run --locked -p oblikovati-head -- --frames 3 "path/to/part.opd"
cargo run --locked -p oblikovati-head -- -- "-literal-name.opd"
```

Frame options can precede or follow paths. Both `--frames N` and `--frames=N`
are accepted, as are the Go-compatible `-frames` forms. Zero keeps the head
running. Duplicate frame options, unknown flags, empty operands, and paths that
the UTF-8 document API cannot represent fail before a window opens. Standalone
help and version do not require a display.

Paths open in command order with `visible: true`; the last becomes active.
The first failure stops later opens and names the failed path. Both startup
failure and loop exit stop owner admission before window retirement. Existing
open documents remain owned for recovery; startup does not rewrite source
files. A bounded frame run is a test exit, not a save or discard command.

## Acceptance

Named fake callers verify the API method and DTO, visible-open order, failed-open
short circuit, and owner/window retirement order. Parser tests cover literal
paths, spaces, option placement, invalid text, and duplicate flags. Executable
tests open a native file and report a missing path. Native owner tests open two
stored prisms, check the active document, capture rendered solid pixels, verify
retirement, and compare the source bytes after the run.

Local Linux acceptance uses the test toolchain in [RELEASING](../../RELEASING.md).
The [native stress report](diagnostic-startup-native-crash.md) records the
validation-layer failure found during acceptance. The full local gate passed for all 12 crates, including
production line coverage above 80% in each crate, native tests, documentation
tests, formatting, Clippy, Markdown, SPDX, and workflow checks. Production
duplication is 0.58%. No remote test, push, or PR is part of this change.

## Remaining work

This adds executable startup to the existing Rust file lifecycle. It does not
establish a recent-files menu, file dialog, save prompt, or editor. Dear ImGui
composition, complete feature coverage, and foreground MCPBridge control remain
open. Exchange codecs belong to Martensite under
[ADR-0018](../decisions/ADR-0018-exchange-format-ownership.md).
