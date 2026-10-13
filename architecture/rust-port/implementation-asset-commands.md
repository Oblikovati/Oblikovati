# Typed asset create and update commands — partial R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
The source behavior is `addin/router/material.go`, `app/materials.go`, and
`model/material/library.go` in the Go application.

The file host now serves the four canonical material and appearance create/update
methods from the released API. The script and native add-in inventories include
these routes. No contract records or method names are redeclared.

Reads and edits share one scoped resolver: document, project, then built-in.
Creation copies all properties into a project definition. IDs follow the Go
lowercase slug and numeric collision suffix rules, including repeated separators
and non-ASCII names. All visible scopes participate in collision detection.
Creation requires a selected project with an explicit design-data directory.

Updates preserve the resolved identity and source. Built-in updates return the
unchanged asset, as in Go. Project edits use atomic persistence before adoption;
write errors reach the caller. Document edits use the existing transaction owner,
so veto, undo, redo, abort, and save/reopen retain normal document semantics.
The reply is encoded before any state change. Project storage does not create
missing design-data directories; project setup remains separate work.

Tests use typed API clients and named memory filesystems. They check stable IDs,
copy properties, project write/reload, document history and reopen, builtin
no-op behavior, source spoofing, unknown assets, empty names, malformed requests,
invalid density, failed writes, and retry identity. A real CLI Lua child edits
large physical scalars and appearance groups, saves them, and verifies the saved
package. It also refuses a rounded integer document ID. The normal local check is
`make gate`; focused checks use `cargo test -p oblikovati-automation`.

Assignment commands, native asset editors, project setup controls, and viewport
refresh from changed asset properties remain open. This is partial application
port evidence, not R7 completion or foreground MCPBridge acceptance.
