# Behavior ledger and native receipts

The source inventory contains one row per scenario declaration. A scenario
outline is one declaration; its examples are separate executable cases. The
classification file is reviewed input. The generated behavior ledger combines
both and retains unclassified, blocked, stale, and orphaned work. It is not a
count of bindings or diagnostic runs.

Use `python3 -B tools/bdd/behavior_cli.py report` to print verified and total
counts by area and milestone. Milestone totals include only assigned scenarios;
unclassified scenarios may belong to any milestone. `generate --output PATH`
writes the combined ledger. Both commands restore the pinned source into a temporary checkout when
no checkout path is supplied. `check` also requires the checked-in ledger to
match the generated ledger and all R0 classification and evidence fields to be
present. The full gate runs this check. It is expected to fail while those
fields remain incomplete.

To run one reviewed scenario and prepare its receipt for visual review, use:

```sh
python3 -B tools/bdd/capture_cli.py \
  --checkout /absolute/path/to/scenario-checkout \
  --scenario-id scenario:FULL_ID \
  --output architecture/rust-port/receipts/UNIQUE_RUN \
  --renderer "Vulkan with Lavapipe under Xvfb"
```

Run the command on a configured display, for example under `xvfb-run`. It runs
the native test, checks the source and application inputs before and after,
checks all reported steps and cleanup, and retains the report and capture in a
new directory. If the run produces multiple PNG files, pass `--capture` with
the intended filename. The command does not change the classification: inspect
the capture and the result before updating `last_receipt` and marking a scenario
verified. A successful receipt alone does not close a milestone.

Use `python3 -B tools/bdd/behavior_cli.py backlog` for one entry per feature
file, ordered by the earliest milestone assigned to one of its scenarios.
Each entry counts verified, blocked, unclassified, stale, and incomplete R0
baseline rows. Its milestone counts include an `unassigned` group. Each feature
also reports the number of rows missing each baseline field. An unbound or
changed source hash makes every field incomplete. These counts do not verify
the quality of the evidence. A mixed feature can
span milestones; the sort order is not a feature-level assignment
or proof that its dependencies are complete. Unassigned files sort last until
their scenarios have reviewed classifications.

A classification records the scenario ID, the source revision and feature
hash, milestone, owner, target module, acceptance test, issue links, state, and
last receipt. R0 also requires the Go reference and test premise, API revision,
kernel capability and integration information, platform profiles, Rust tests,
oracle, and expected result. Explain an inapplicable field instead of leaving
it empty. Set `baseline_source_sha256` to the reviewed feature-file hash;
missing, invalid, or changed hashes leave the baseline incomplete. A renamed
scenario receives a new ID; review its orphaned former
classification. A changed feature invalidates the prior receipt even when the
scenario name stays the same.

A verified ordinary scenario requires a receipt with this shape. Artifact paths
are relative to the repository root; each hash is the SHA-256 of that file.

```json
{
  "kind": "native-acceptance",
  "scenario_id": "scenario:<source-id>",
  "scenario_revision": "<pinned-source-commit>",
  "source_sha256": "<feature-file-sha256>",
  "application_commit": "<tested-app-commit>",
  "api_revision": "<locked-api-commit>",
  "kernel_revision": "<locked-kernel-commit>",
  "runner": "application_scenarios_use_the_native_window",
  "exit_code": 0,
  "environment": {"os": "Linux", "renderer": "Vulkan"},
  "cucumber": {"path": "path/to/cucumber.json", "sha256": "<sha256>"},
  "source_report": {"path": "path/to/source-revision.json", "sha256": "<sha256>"},
  "capture": {"path": "path/to/capture.png", "sha256": "<sha256>"}
}
```

The classification's `last_receipt` has the same `path` and `sha256` shape.
The checker requires Python 3.11 or newer, a clean pinned checkout before and
after the run, matching application inputs, a native runner, a PNG capture, a passing cleanup hook,
and every source step in the Cucumber report. It verifies feature and rule
backgrounds separately. For an outline, it requires every inventoried example
line, expanded name, and expanded step sequence to pass, including cleanup.
It refuses outlines with unchecked step tables or docstrings. It does not
treat a PNG alone, a diagnostic report, or a
source binding as acceptance. Visual inspection and the other milestone exit
checks still apply.

When application inputs change, the ledger marks the receipt stale and keeps
its historical link. Missing or altered receipt artifacts are errors, including
for stale evidence. Input comparison excludes the README, Rust-port ledger and
receipt documents, and the BDD and build-driver tools. The native test runner,
application crates, lockfile, and other repository inputs remain in scope. The
current inventory and evidence checkers run again for every report.

The tracked-input comparison ignores Git replacement objects, inherited Git
configuration, and external diff commands (#4016). This does not bound the
runtime or establish build provenance. Native receipts remain stale under issue #3978 until that evidence is established.
