# BDD workflow for the Rust port

MCAD-Scenarios is the behavior benchmark. Use the pinned snapshot for runs and
review live source updates in the weekly sync. The Go implementation in
`../Oblikovati` is the architecture and implementation reference. Follow
milestone order, with application and sketch scenarios as acceptance evidence. Do not treat a source
binding, a unit test, or an API-only test as proof of a native UI scenario.

## Restore the pinned source

The source directory is edited without Git. The pinned commit is a local snapshot,
not an upstream revision. Without `OBK_BDD_ROOT`, the native runners and binding
inventory restore and verify a temporary checkout of the pinned bundle. The
checkout is removed when the test thread ends. A gate run requires the pinned
revision, a clean checkout and unchanged content; it rejects diagnostic and
feature or scenario filters. The gate still has its documented accepted sketch
feature list and is not a full sketch-area run.

For a reproducible focused run, restore and verify the checkout explicitly:

```sh
git clone -c core.autocrlf=false --branch snapshot \
  "$PWD/test-utilities/benchmarks/scenario-snapshots/mcad-scenarios-2026-11-16-b576445.bundle" \
  /absolute/path/to/scenario-checkout
python3 tools/bdd/verify_snapshot.py /absolute/path/to/scenario-checkout
export OBK_BDD_ROOT=/absolute/path/to/scenario-checkout
```

Keep the checkout outside the live source directory. Explicit `OBK_BDD_ROOT`
overrides also support unversioned or dirty diagnostic sources outside the
gate; their identity is reported and they are not pinned acceptance evidence.
Each native run records `source-revision.json`, including the commit and dirty
state before and after the run. A source change during a run invalidates its results. If the source is
not a Git checkout, the report records a null commit and dirty state.

Outline receipts keep one image per example, named from the feature path and the
example's source line. The checker requires and hashes every image; old outline
receipts with a single capture need a new native run. Ordinary receipts keep the
single-capture format. The `--capture` option applies only to ordinary scenarios.

## Discover and triage changes

During the weekly source sync, compare the live directory with the inventory:

```sh
python3 tools/bdd/inventory.py /mnt/vininas-projects/MCAD-Scenarios \
  --baseline architecture/rust-port/bdd-source-inventory.json
```

The inventory compares file content hashes, not modification times. It reports
added, changed, and removed files. Scenario counts are declarations, not
executable test counts: an outline can produce multiple cases. The command does
not execute tests and does not claim a pass or failure.

Review every changed file, including edits that keep the same scenario name.
Compare its requirements with the implementation and previous evidence. A
change invalidates all previous results for that file until it is run again.
If a file is renamed, treat the old path as removed and the new path as added.
If a source correction conflicts with observed behavior or other scenarios,
record the conflict in `scenario-defects/` and the relevant issue; do not
silently weaken an assertion.

Each live feature folder has a `BLOCKERS.md` for the scenario author. Record
only issues the author can fix in the scenario source, such as invalid syntax,
contradictory steps, or unclear fixture requirements. Do not include Rust
implementation details or ask the scenario author to fix the Rust port. Track
those gaps in the Rust ledger and GitHub issues. Recheck a reported source
defect before removing it. An empty blocker list does not prove that a feature
passes.

After triage, update the pinned snapshot, source inventory, and reviewed
classifications together. Preserve old receipts as historical evidence when a
source hash changes. The discovery inventory is not evidence that features are
implemented. Check the live source again before committing: it can change while
work is in progress.

A reviewed assignment can use the new source hash while its baseline remains
historical. Set `baseline_source_sha256` to the source that baseline describes;
the baseline-field check treats a mismatch as missing evidence. Keep the old
baseline details and review record for inspection. Update the baseline hash
only after checking the evidence against the current source.

Run `python3 -B tools/bdd/behavior_cli.py report` after updating the pin. This
regenerates the scenario inventory from the pinned checkout and compares every
field, including inherited capture tags, with the checked-in inventory. A
successful discovery inventory alone does not check those fields. Regenerate
the behavior ledger from the same inputs before committing.

## Select and implement work

Choose the next ready item in milestone order. Record its scope, problem, and
dependencies in a GitHub issue. Assign each scenario in the classification
file and link the issue. Run `python3 -B tools/bdd/behavior_cli.py queue`
to see open scenario work grouped by milestone and feature. The queue includes
blocked work, stale evidence, and missing R0 baseline fields; it does not assert
that dependencies are ready. Unclassified scenarios are counted separately.
Use `behavior_cli.py backlog` for feature-wide totals.
Use the Go source as a design reference and implement through the shared Rust
host operations. Add the appropriate model and host tests, then drive the native
window with mouse and keyboard for UI behavior. Close test windows afterward.

Use these states:

- **Not started**: the requirement is known but not being implemented.
- **In progress**: implementation or focused verification is underway.
- **Blocked**: identify the concrete external dependency or scenario conflict.
- **Verified**: native scenario evidence is recorded for the current inputs.
- **Needs rerun**: source, application code, dependencies, or relevant runtime
  changed after the recorded run.

The source tag `@capture_only` excludes a recorded observation until its
fixture is available. The tag `@capture_host_only` blocks execution until the
named capture host and its installed fixture can be verified; there is no
host-attestation bypass yet. Neither state is a passing scenario. The native
runner reports them as not executed, and the diagnostic report records their
classification.

Do not mark a full feature verified from a subset of its scenarios. Record the
exact filter and cases run. Unbound, skipped, filtered, or unexecuted cases are
not passes. Use full area runs to report area totals; the default sketch gate
only selects its accepted feature list, so it is not a full sketch run.

The receipt capture CLI builds the selected commit from a temporary export with
a new temporary target. It deletes both after the run. This excludes ignored checkout files from the initial build tree and prevents
reuse of previous target artifacts, but it does not bound
compiler settings or external build inputs. Captures remain historical evidence
until issue #3978 is resolved. A full rebuild requires time and disk space.

The CLI copies the registry index into an offline Cargo home. It copies locked
registry archives after checking their SHA-256, and stages Git databases for
locked commits and recursive submodules with Git fsck. Cargo uses fresh source
and checkout directories. The copied index is not authenticated.

The source Cargo config may contain only the current xtask alias and Linux
rpath setting. Other settings are refused until reviewed. All exported Cargo manifests
cannot contain patch or replace tables or paths outside the export. These files
and the lockfile are checked before and after the build. Tools, dependency build scripts, host
files, runtime settings, and changes during the build remain unproven. All locked
registry archives and Git commits, including submodules, must be available in
the local cache. The build needs temporary disk space for these inputs.

Native capture also builds its inventory generator from the selected commit in
a separate temporary offline target. `OBK_SCENARIO_LEDGER_TOOL` does not select
the generator for capture. This adds another build and needs the tool's locked
dependencies in the local cache. The host compiler and caches remain unproven.

## Record test evidence

Each run uses a unique default report directory under
`target/benchmarks/application`. For a known path, give each run its own absolute
`OBK_BDD_REPORTS` directory. An occupied report directory is refused; prior
artifacts are never removed or overwritten by a new run. Use a separate absolute
`CARGO_TARGET_DIR` for each concurrently active worktree. A shared target can
reuse a test binary built from another checkout; discard that run as acceptance
evidence if the binary provenance is uncertain. Keep the target until its runs
finish, then remove task-owned temporary targets when they are no longer needed.
Retain a checked receipt and its artifacts as described in
[behavior receipts](behavior-receipts.md), including:

1. Feature file path and SHA-256 before and after the run. If the hashes differ,
   discard the run as current evidence and rerun against stable source.
2. Rust application commit, relevant uncommitted changes, Martensite dependency,
   operating system, display and renderer setup.
3. Exact command, feature and scenario filters, date, pass/fail/skip counts,
   and the saved Cucumber JSON and JUnit report locations.
4. Remaining gaps, failures, or links to scenario and kernel defect records.

For a focused native run, set `OBK_BDD_FEATURE` to the feature folder and, when
needed, `OBK_BDD_SCENARIO` to a scenario-name substring. The latter is a filter,
not an exact identifier; inspect the report to identify every executed case.
For example, after configuring the local Vulkan runtime and an Xvfb display:

```sh
export OBK_BDD_ROOT=/path/to/MCAD-Scenarios
export OBK_BDD_REPORTS=/absolute/path/to/unique-report-directory
export CARGO_TARGET_DIR=/absolute/path/to/this-worktree-target
export OBK_BDD_FEATURE=pattern-rectangular
cargo test --locked -p oblikovati-head --lib \
  sketch_scenarios_use_the_native_window -- --nocapture
```

On Linux, use the private desktop runner for Main Window Frame. It starts
Xvfb, Openbox, tint2 and AT-SPI and enables accessibility in the test session.
Install those tools and the Python GI bindings for AT-SPI first.

```sh
OBK_BDD_ROOT=/path/to/MCAD-Scenarios \
OBK_BDD_REPORTS=/absolute/path/to/unique-report-directory \
VK_DRIVER_FILES=/usr/share/vulkan/icd.d/lvp_icd.json \
XMODIFIERS=@im=none \
  crates/head/tests/accessibility_atspi/frame_run.sh
```

The Linux docking-system restart scenario also starts the desktop executable
twice. Build `oblikovati` from the same source as the test, and set
`OBK_BDD_DESKTOP_BINARY` to its absolute path. Run with an X11 window manager,
a 1440 × 900 display, XTest, and ImageMagick `import` and `compare`. The
scenario writes the before and after images and saved layout to the report.
It uses an isolated settings directory and closes both application processes.

The report directory is local output, not durable evidence. Preserve its
required artifacts in the repository and link them from the receipt. If reports
are lost, the run remains historical context, not current acceptance evidence.
Do not sum reports from different revisions, changed scenario files, or
overlapping filters as a current total.

## Check the capture pipeline

Use the real native capture check to exercise report reservation, the capture
CLI, receipt publication, and receipt validation together. Select a new output
directory under `architecture/rust-port/receipts` and run on a configured
display. For example, on Linux with Lavapipe:

```sh
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a -s '-screen 0 1600x1200x24' \
  python3 -B tools/bdd/live_capture_check.py \
  --output architecture/rust-port/receipts/UNIQUE_RUN \
  --renderer 'Vulkan Lavapipe under Xvfb'
```

The check restores a temporary pinned checkout, runs the native scenario,
publishes the receipt, and validates it again. Review the capture before
marking the scenario verified. The optional `--scenario-id` selects another
reviewed declaration.

## Diagnose a feature

Set `OBK_BDD_DIAGNOSTIC=1` on the focused command to write `gap-report.json`.
The runner uses the native window and continues after a failed or unbound step.
It marks subsequent step results as tainted because the earlier failure may have
changed the world. The report includes the exact feature text and is discarded
if that source changes during the run. Ambiguous bindings and window setup
failures have their own classifications. A fixture tag alone does not establish
that a fixture is missing; investigate fixture failures before classifying them.

Diagnostic results are for sizing work. Run the normal command without
`OBK_BDD_DIAGNOSTIC` to collect acceptance evidence.

To select a solid-model feature, set `OBK_BDD_AREA=solid-model` and use
`application_scenarios_use_the_native_window` with `OBK_BDD_FEATURE`. The
runner uses the same native window and step registry. Unbound steps still fail
normal acceptance. Unset `OBK_BDD_AREA` to restore the application default;
sketch features use the separate sketch runner.

## Inventory scenario declarations

After restoring and verifying the pinned checkout, generate the source inventory:

```sh
cargo xtask scenario-ledger "$OBK_BDD_ROOT" \
  e76cb749ed43c73a91c24854057da7c890041f51 > /absolute/path/to/source-inventory.json
```

The command requires a clean checkout at the requested commit and reads its
features twice. IDs use the feature path and scenario name; a rename creates a
new ID and requires review. Each outline counts once as a declaration, not as
its expanded executable cases. Invalid Gherkin files retain provisional
scenario declarations marked blocked with their parser error. The checked-in
`behavior-source-inventory.json` is a source inventory: its null milestone,
owner, and receipt fields do not establish implementation or acceptance.
Do not overwrite future behavior classifications with this raw inventory.

## Feature-by-feature run inventory

Work on one feature until all its scenarios pass. Before implementation and
verification, read its `.feature` file and its available `FUNCTIONAL.md`,
`VISUAL.md`, `FIXTURES.md`, `BLOCKERS.md`, and reference images. Check fixture
and visual assumptions against those files. Record the hashes of supporting
files used for the run; a changed supporting file requires review as well.
To inspect the live source and its recorded full-feature runs, use:

```sh
python3 -B tools/bdd/feature_progress.py /mnt/vininas-projects/MCAD-Scenarios
```

The report includes the current SHA-256 of each `.feature` file, its scenario
declaration count, and the number of scenarios passed in the recorded run. A
feature is `green` only when the source hash matches, the report hash matches,
every scenario and its source steps are present, and every step and cleanup
hook passed. A source change becomes `source_changed`; a change to a recorded
supporting file becomes `support_changed`. A removed feature is listed separately. A `blocker` on a hash-matched result reports the feature as
`blocked`. Matching scenario passes still count unless the record lists them
as unproven. For outlines, the inventory
checks every example against a parsed scenario snapshot when its feature hash
matches. A run may provide a separate parsed inventory and its SHA-256 in
`scenario_inventory` and `scenario_inventory_sha256`; generate it with the
standard `cargo xtask scenario-ledger` from a clean snapshot of the source.
For a local unversioned source, `cargo xtask scenario-local-ledger "$OBK_BDD_ROOT"`
generates a double-read snapshot labeled `unversioned`. It supports the feature
progress inventory but does not establish strict verified acceptance.
Changed outlines without a matching parsed inventory need further review.
Ordinary scenarios with backgrounds, rules, tables, or docstrings require a
parsed inventory with `arguments_recorded` set and the native runner's
`scenario-arguments.json`. Record the latter as `scenario_arguments` and its
SHA-256 as `scenario_arguments_sha256`. The runner captures parsed arguments
before each scenario and binds the capture to the Cucumber report hash. The
inventory compares every argument and background
against the source snapshot, then checks results and cleanup. Old reports and
source snapshots without argument evidence remain uncounted. Outline arguments
retain their separate, stricter review rule.

After running a whole feature, retain its Cucumber report and other artifacts
under `receipts/`, then add the source hash, report path and hash, tested commit,
command, and environment to `feature-run-results.json`. To retain runs for
several versions of one feature, put the records in that feature's
`by_source_sha256` object and use each feature hash as its key. The report
selects the record that matches the source it reads. Check the feature hash
before and after the run. A green entry means that the complete recorded run
passed at the listed application commit; it does not prove later application
commits. It is distinct from the stricter verified acceptance ledger and its
build-provenance requirements.
