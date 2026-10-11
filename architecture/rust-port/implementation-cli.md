# Headless CLI — partial R2

Tracking: [#3626](https://github.com/Oblikovati/Oblikovati/issues/3626).
The `oblikovati-cli` workspace package provides a real Linux executable with the
same name. Its command runner accepts injected package input and output, while
the executable uses native files and reports failures on stderr with exit code 1.

## Implemented command inventory

| Command | Behavior |
| --- | --- |
| `new <type> <path> [--seed]` | Create and atomically save an unseeded document through the file owner; see [creation evidence](implementation-cli-new.md). |
| `info <path> [--json]` | Decode a schema-2 envelope and report schema version, document kind, and display name without executing its recipe or writing the file. |
| `version`, `--version` | Report embedded product version and application commit. |
| `help`, `-h`, `--help` | Report available commands and the current implementation scope. |

`info` retains the Go JSON shape: `schemaVersion`, `type`, and `name`. Its Boolean
JSON flag can appear before or after the path. Document kinds come from the API
crate. Native path operands stay OS strings. Read, decode, usage, and output-stream
failures propagate to the caller rather than producing success output.

`open`, `save-as`, `import`, `export`, `export-flat`, `script`,
`generate-assembly`, and `check-updates` report that they are not yet ported.
They do not pretend to execute the available partial model as a complete CAD
document. Typed automation routing, full recipe reconstruction, command edits,
and geometry remain outstanding R2/R3 work.

## Build identity and evidence

The build script embeds the complete `OBLIKOVATI_BUILD_MANIFEST` JSON and declares
the manifest path and environment variables as rebuild inputs. Version output
uses its product version and application SHA, rather than the Cargo tools version.
Without a supplied manifest, the binary explicitly reports the package version
with a `-dev` suffix and an unknown commit. Manifest identity errors fail the build.
No current wall-clock timestamp is introduced by the build script.

Six command tests cover all document kind names, JSON/text output, flag position,
invalid and unported commands, input/output failures, and build identity parsing.
A native process test checks successful JSON inspection, unchanged file bytes,
missing-file stderr/exit status, and version output. These are local Linux CLI
checks; they do not certify GUI, geometry, packaged deployment, or other platforms.

The Go `TestInfoJSONIsParseable` reference test passes at the fixed baseline.
A separate Linux build supplied with a synthetic packaging manifest reports
`0.26010.2.7` and that fixture's commit through `--version`. This checks manifest
embedding and product-version selection, not a published release identity.
