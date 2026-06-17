# Wiki sources

These Markdown files are the version-controlled source of the
[Oblikovati GitHub wiki](https://github.com/Oblikovati/Oblikovati/wiki). The wiki itself is a
downstream mirror — manual edits there are overwritten on the next publish.

- Static pages live here (`Home.md`, `_Sidebar.md`, …) and are copied verbatim.
- **`Command-Manual.md` is generated** by `cmd/command-manual` from the built-in command
  vocabulary in `app/cmdline`; it is not stored here.
- **`Lua-Scripting.md` is generated** by `cmd/lua-manual` from the wire API (`api/wire` +
  the `api/client` `mcp:summary` annotations) and the `script/examples` library; not stored here.

`scripts/publish-wiki.sh` assembles the static pages plus the generated manual and pushes
them to the wiki. The `Wiki` GitHub workflow runs it on every merge to `develop`.

To preview locally:

```sh
go run ./cmd/command-manual /tmp/Command-Manual.md
go run ./cmd/lua-manual "$(go list -m -f '{{.Dir}}' oblikovati.org/api)" /tmp/Lua-Scripting.md
```

This `README.md` is repo-only and is never published to the wiki.
