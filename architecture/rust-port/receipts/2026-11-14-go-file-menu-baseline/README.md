# Go File menu baseline

Go checkout: `aa8297d1b` (`go-legacy`). The File menu source is
`head/ui/chrome_menubar.go`, `drawFileMenu` and `drawOpenRecentMenu`.
It uses native menu items, with New Part, New Assembly, and New Drawing as
separate entries. It does not implement the scenario's New and Save As
submenus or its recent-documents region. The following tests cover related
state and dispatch only; they do not drive or verify the File menu.

```text
$ go test ./app -run '^TestOpenDocumentFromMRUHonorsVetoAndRecords$' -count=1
ok  oblikovati.org/app  0.005s

$ cd head && xvfb-run -a go test ./ui -run '^(TestFileDialogConfirmReturnsActionForMode|TestQATDispatchDisabled)$' -count=1
ok  oblikovati.org/head/ui  0.006s
```

The head compile emitted a warning in the vendored ImGui Vulkan backend. The
MRU test uses a fake document store and checks a path, deduplication, and a
veto; it is not a menu rendering or selection test. The file-dialog test
checks its state machine without the cgo draw path. The quick-access test
checks dispatch state and does not exercise the File menu. These premises
are valid for the stated unit scopes but cannot prove the scenario UI results.
No exact Go test for the eight File menu scenarios was found in the checked
Go test sources. The pinned scenario and Rust native receipt provide the
acceptance evidence; they do not establish visual parity with Go.
