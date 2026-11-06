# Native CLI document creation

Tracking: [#3751](https://github.com/Oblikovati/Oblikovati/issues/3751).
The file owner and storage boundary remain ADR-0003 and ADR-0004.

## Creation and files

`oblikovati-cli new <type> <path>` creates an unseeded part, assembly, drawing,
or presentation. Kind parsing trims surrounding spaces and ignores case. The
API owns the numeric kind and extension constants. Missing or different
extensions are appended; matching extensions retain their original case.
Leading-dot filenames follow Go `filepath.Ext` rather than Rust's dotfile rule.
Linux native path bytes remain intact through canonical lookup and atomic save.

Creation, identity issuance, snapshot preparation, and save use the existing
FileWorkspace. The saved package has its first save identity and the current
complete recipe. Success output follows successful storage. Read-only library
execution remains available; the native binary uses `run_with_files`, which
injects document access and GUID generation. Output and storage errors propagate.

Part `--seed` is refused before file access or identity allocation until the
Martensite extrusion path is available. Non-part seed emits the Go note and
creates the ordinary document. Boolean seed flags accept the Go spellings and
can appear before or after operands; a later false value clears an earlier flag.
The seeded 4-by-3-by-5 solid remains R1/R3 work.

## Linux evidence

Named fake file and GUID sources test all kinds, extensions, Boolean options,
invalid arguments, identity failures, each atomic-write phase, output failure,
and unchanged destination bytes after failure. Native process tests create and
inspect all four kinds, load their full packages, preserve non-UTF-8 filenames,
replace an existing file with one complete package, and check failure exit codes.

Go `TestNewWritesPackageThatInfoReports`, `TestNewCreatesEachKindWithItsExtension`,
and `TestNewAppendsPackageExtension` pass at source revision `aa8297d1`.
[Saved-title regression evidence](implementation-save-names.md) covers the shared
manifest fix found by the native creation tests. Acceptance requires the full
Linux `make gate`. No geometry, GUI, installation, or other-platform completion
is claimed.
