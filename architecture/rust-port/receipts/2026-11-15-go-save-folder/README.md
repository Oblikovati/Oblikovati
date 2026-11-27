# Go reference for Save As folder navigation

At Go application commit `aa8297d1b0ac33c024f19e54821536d76e8a447a`,
the following UI state tests passed on Linux under Xvfb:

```sh
cd ../Oblikovati/head
xvfb-run -a go test ./ui -run '^(TestFileDialogChooseDirectoryAndFile|TestFileDialogSaveJoinsDirectoryAndDefaultExt|TestFileDialogCancelYieldsNoAction|TestFileDialogOpenForReArmsAndClears)$' -count=1
```

These tests check entering a directory, forming a save path, cancelling an Open
dialog, and rearming dialog state. They do not drive the native UI or prove the
Back navigation and proposed-name retention in the two pinned scenarios. The
Go dialog has an Up control; this check did not find an equivalent Back control.
The Rust native evidence is in `../2026-11-15-save-subfolder/`.
