# Development rules

## Code style guidelines

- Functions: 4-20 lines. Split if longer.
- Files: under 500 lines. Split by responsibility.
- One thing per function, one responsibility per module (SRP).
- Names: specific and unique. Avoid `data`, `handler`, `Manager`.
  Prefer names that return <5 grep hits in the codebase.
- Types: explicit. No `any`, no untyped functions.
- No code duplication. Extract shared logic into a function/module.
- Early returns over nested ifs. Max 2 levels of indentation.
- Exception messages must include the offending value and expected shape.
- Match the surrounding file's conventions: naming, comment density,
  module structure. Comments explain *why*, never restate *what*.
- Small, scoped, behavior-preserving changes. No adjacent feature work,
  no opportunistic refactors, no speculative abstractions or new public
  surface without a shipped caller, persisted data, or an explicit
  requirement.
- Record Rust port feature evidence in `architecture/rust-port/implementation-*.md`.
- Remove throwaway experiment implementations and their branches/worktrees once the
  investigation concludes; retain findings and measurements in `architecture/rust-port/`,
  not dormant experiment code. Preserve unrelated unfinished feature work.
- No dead code or half-built public surface. Future work is documented in
  `architecture/rust-port/` design notes, not shipped as unreachable stubs.
- Prefer explicit fallbacks over `unwrap`, `expect`, or `unreachable!` in
  runtime paths; panics are acceptable in tests only.
- Workspace lints forbid unsafe by default. Existing platform and FFI modules have explicit audited exceptions; do not add unsafe outside a reviewed boundary.
- Typed boundaries are load-bearing: IDs and dialects are parsed/normalized once and reused.
- Keep CLI commands thin: parse args, resolve config once, call typed
  library functions, render output.
- Keep your own comments. Don't strip them on refactor — they carry
  intent and provenance.
- Write WHY, not WHAT. Skip `// increment counter` above `i++`.
- Docstrings on public functions: intent + one usage example.
- Reference issue numbers / commit SHAs when a line exists because
  of a specific bug or upstream constraint.

## Testing instructions

- Tests run with a single command: `cargo test`. `make gate` runs the full
  gate over all discovered workspaces; run it before the first push and before each PR and batch.
- Every new function gets a test. Bug fixes get a regression test.
- Mock external I/O (API, DB, filesystem) with named fake classes,
  not inline stubs.
- Add focused regression tests for every bug fix and behavior change.
  Parsers, ID derivation, and math especially.
- Filesystem tests use temp dirs or injected roots (`tempfile`); never
  depend on the real user home directory being writable.
- Tests must be F.I.R.S.T: fast, independent, repeatable,
  self-validating, timely.
- Integration / E2E / BDD tests are placed in tests

## Formatting

- Use the language default formatter (`cargo fmt`). Don't discuss style beyond that.

## Logging

- Structured JSON when logging for debugging / observability.
- Plain text only for user-facing CLI output.

## Security considerations

- **Never commit secrets.** Keep real tokens out of docs, fixtures, and
  tests. No secret scanner runs in CI yet.
- **Dependency policy:** run `cargo deny --all-features check` and
  `cargo audit` when you add or update a
  dependency; they are not CI gates yet. Do not add dependencies without
  checking the project doesn't already have the capability, and match
  existing versions/idioms.

## Project maintenance rules

- **CHANGELOG is a merge gate.** Any change affecting user-visible
  behavior, installation, supported platforms/agents/providers,
  deployment, env/config, or public tool/admin surfaces must add a
  `CHANGELOG.md` entry under `## [Unreleased]` (correct Keep-a-Changelog
  heading — `Added`/`Changed`/`Deprecated`/`Removed`/`Fixed`/`Security` —
  past-tense, trailing `(#NNN)` reference) and update the relevant
  README/docs references in the same commit. Internal refactors and
  test-only churn are exempt.
  File each entry under the heading that matches its semver impact. Released
  sections are frozen. `cargo xtask release-request <channel>` prepares a release
  request after its preflight; it does not check changelog heading semantics.
- **Reclaim build storage after every release — run `cargo clean`.**
  Once a release is tagged and its artifacts are published, run
  `cargo clean` (and `cargo clean` in each release worktree / extra target dir
  you created, then remove finished `git worktree`s). This is a mandatory
  post-release cleanup step, not optional housekeeping. It also removes the
  rehearsal packages in `target/dist/`. During normal development, a weekly
  `cargo clean` is the lighter-touch equivalent.
- **No version bumps or release tags without explicit user approval.**
  Do not bump crate/package versions automatically.
- **Test only changed or affected packages.** Run a package's suite when its own
  source, tests, build inputs or dependencies can change its behavior. Include
  transitive dependents; skip unchanged, unaffected packages and base dependencies
  that merely need compiling. Classify the actual impact of manifest, lockfile
  and build-configuration edits before selecting suites. Dependency impact can
  require tests without requiring a version bump. Release validation still runs
  the required platform matrix, using changes since the previous release to
  determine package scope.
- **Non-negotiable quality bar.** No malicious or obfuscated code, no
  regressions (add tests when unsure), no stale docs, no security or quality
  degradation. Any one blocks a merge regardless of approval; when in doubt,
  double-check and live-test with the automated checks.
- **PR evaluation:** audit each PR (pros, cons, recommended fix) and carry
  approved tickets through yourself: resolve soft blockers (rebase or base
  retarget, missing tests/docs/CHANGELOG, small fixes and easy decisions) as
  maintainer commits; ask the author only about a hard blocker you cannot
  resolve. Once the maintainer approves a batch, push each green task to its
  branch as it finishes. Version bumps and tags always need explicit
  approval.
- **Review baseline:** always use `rust` for code review and pin its merge-base
  with the reviewed branch. Do not ask the user to choose the routine baseline.
