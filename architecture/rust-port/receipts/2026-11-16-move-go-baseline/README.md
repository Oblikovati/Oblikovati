<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Go Move baseline for history scenarios

On Go revision `aa8297d1b0ac33c024f19e54821536d76e8a447a`, this command passed:

```sh
go test ./app -run '^TestSketch(Move|Copy)Tool$' -count=1
```

It returned `ok oblikovati.org/app`. The Go checkout had untracked audit
Markdown files; no tracked changes were reported. The tests translate a line
and check the original line after Copy. They do not exercise ellipse history,
UI interaction, or scenario acceptance. The Rust history diagnostics and their
limits are recorded in `../2026-11-16-move-history-4014/`.
