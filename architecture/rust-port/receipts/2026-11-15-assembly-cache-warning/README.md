# Assembly save warning live check

Date: 2026-11-15. API commit: `21b18a952fa7548a31e7eac338f6c78a32e5cfe1`.
The host was the `agent/express-cache` worktree at parent commit
`eccbed3849e7b9e8d914d362d3b31b0bdbcfc31f`, with uncommitted cache and warning
changes and a local Cargo patch to that API commit. The host binary was removed
with its build target before its checksum was captured. This receipt does not
identify a final host commit.

On Linux, the foreground host ran in Xvfb with Mesa lavapipe. The bridge was the
existing Go shared library, SHA-256
`2942bdba7a756d20ea17f24fb18e9eb41f495e20b1d259caf9d28d77e237b20b`.
The launcher used a private add-in directory, loopback port 17893, and this
assembly input:

```yaml
schemaVersion: 2
documentType: 2
model:
  occurrences:
    - name: Missing:1
      component: missing-component.opd
```

The exact launcher command was:

```sh
xvfb-run -a -s '-screen 0 1280x800x24' bash /tmp/obk-express-live/run.sh
```

The copied `run.sh.txt`, `driver.go.txt`, and `key.py.txt` contain the scripts used. The driver
called the real MCPBridge list, save, and capture tools; XTest sent Ctrl+S to
the actual application window. `transcript.txt` records their output. The
1440 × 900 capture is `window.png`, SHA-256
`4a4a824bafcbafef6b6d9c8456453c0530dfcdfa9a345f780db08d664b8da958`.
The screenshot was inspected: the browser shows an unresolved occurrence and
the status line shows the cache warning, with its detail clipped at the right
edge. The app and MCP listener were stopped after capture. This run did not
open the Log Panel, exercise Express open, or test Load Full.

The API `make gate` passed at the API commit above. `api-gate.log.gz` contains its
output. Go statement coverage was 84.5%; `api-rust-coverage.log.gz` reports 97.73%
Rust line and 95.71% function coverage. The coverage run and cross-builds were
on the prior commit; the only later changes were comments and regenerated
comment text. Go builds and Rust library checks passed for Windows amd64 and
macOS arm64; neither was a native run. The checked-in API comments and constants
were reviewed for platform-specific assumptions. `api-duplication.json.gz` is the
jscpd 4.0.5 result for Go and Rust sources and tests, with minimum five lines
and 50 tokens: 1.97% duplicated lines. The complete API gate emitted unresolved
symbol warnings during documentation generation, but exited successfully.

A separate native regression used `+` → `Log Panel` after the same stale-source
warning and verified the complete warning detail against the rendered text,
ignoring line-wrap whitespace. It passed with one test thread against the published
API revision. This was a native test, not a second MCPBridge capture.

Against the remote API pin, the persistence, session, automation, and CLI suites
passed 1,831 tests and doc tests. The three serial native fixture tests also
passed. Strict Clippy passed for all targets of the five affected crates. The
logs are `obk-express-host-suites.log.gz`, `obk-express-host-native.log.gz`, and
`obk-express-host-clippy.log.gz`. The application `make gate` was not run for this
worktree; the checks here do not establish full repository coverage or a complete
Express-open journey.

The compressed logs and profiles are lossless; use `gzip -dc` to inspect them.

## Combined branch checks

At merged application commit `68fe8fca333794b1b4f58c6d37e47796578d0b74`,
`merged-suites.log.gz` records passing persistence, session, automation, and
CLI tests and doc tests against the published API revision. The three native
assembly fixture tests also passed; see `merged-native.log.gz`.

The document lifecycle methods were then separated to restore the file-size
guard. The same three native tests passed with that split in the worktree before
commit `0a4ad5fed`; `split-native.log.gz` records the run. Documentation lint
passed in that worktree. These checks do not establish Express open or Load Full.
