# Ribbon Appearance evidence baseline

The baseline covers all 18 Ribbon Appearance declarations in snapshot
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The pinned and live feature files
have the same SHA-256. The records bind expected results to the feature
steps and identify the Go reference, Rust tests, and locked dependencies.

The following tests passed on Linux in the Go reference at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`. They cover panel widths and collapse,
icon resolution and button styles, and the system browser handoff. The
appearance, visibility, and reset steps require native desktop evidence.

```sh
# Run from the Go repository.
cd head
xvfb-run -a go test ./ui -run '^(TestPanelsThatFitEverythingFits|TestPanelsThatFitCollapsesFromTheRight|TestRibbonRowWidthCountsDividersOnce|TestRibbonCommandIconsResolve|TestRibbonIconCommandsHaveButtonStyle)$' -count=1
go test ./internal/sysopen -run '^Test' -count=1
```

The baseline records the test premises. Strict acceptance also requires
verified build and runtime inputs under #3978.
