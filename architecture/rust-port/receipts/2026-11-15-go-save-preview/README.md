# Go reference for Save As preview cancellation

At Go application commit `aa8297d1b0ac33c024f19e54821536d76e8a447a`,
the following tests passed on Linux with Xvfb for the UI test:

```sh
cd ../Oblikovati
go test ./app -run '^(TestSaveActiveDocumentNeedsPathBeforeFirstSaveAs|TestSaveActiveDocumentSucceedsAfterSaveAs)$' -count=1
cd head
xvfb-run -a go test ./ui -run '^(TestFileDialogConfirmReturnsActionForMode|TestFileDialogCancelYieldsNoAction|TestFileDialogOpenForReArmsAndClears)$' -count=1
```

The application tests check whether a first save needs a path and whether a
later save succeeds. The dialog tests exercise Save As confirmation, Open
cancellation, and rearming the dialog state. They do not drive the native UI.
A search of the Go `app` and `head` sources found no Save Preview Picture or
preview-source controls. These tests are a partial reference, not proof of the
preview-cancellation behavior. The Rust native evidence is in
`../2026-11-15-preview-cancel-verified/`.
