# Action plan: review of the Rust port plan

Date: 2026-11-13. Scope: `architecture/rust-port/` (`plan.md`, `milestones.md`,
`verification.md`, `bdd-workflow.md`, `bdd-work-ledger.md`), `AGENTS.md`, and
the state of the repository on branch `feat/rust-consolidated` at `a306d76ea`.

This report has four parts: the verdict, the criticism with evidence, the
iteration loop that the agent must follow, and the ordered actions.

## 1. Verdict

The plan is strong on rules and weak on control. It tells the agent what is
forbidden and what evidence is necessary. It does not tell the agent what to
do next, how much to do in one step, or when to stop. The result is high
output with low closure.

| Measure | Value |
| --- | --- |
| Commits on the branch | 1001 in 11 work days (maximum 177 in one day) |
| Commits not on `origin/rust` | 981 (the branch is not on the remote) |
| Commits that refer to an issue | 24 of 1001 |
| Milestones R0 to R9 with a closed exit | 0 of 10 |
| Benchmark size | 195 feature files, 2719 scenarios, 21 992 steps |
| Application feature rows marked Verified | 4 of 24 |
| Sketch and solid-model areas with a full current run | 0 of 2 |
| Step bindings in `crates/head/tests` | 4773 |
| Rust files above the 500-line limit | 20 |
| `bdd-work-ledger.md` | 1395 lines (limit: 500) |
| Local branches / worktrees | 28 / 3 |

The work is real: 13 crates, approximately 140 000 lines of source, and a
native BDD runner. But no gate is closed, and the plan cannot show how far
the port is from completion.

## 2. Criticism

### 2.1 Two plans steer the work, and they disagree

- `milestones.md` gives a dependency order: R0 → R1/R1V/R2 → R3 → R4 → R5 …
- `bdd-workflow.md` gives a different order: application scenarios first,
  then sketch, then solid model.
- The agent follows the second. Most of the recent work is ribbon
  customization, docking, key tips, and Quick Access. That is R7 work. The
  R3 exit (a valid parametric part with a face reference that survives
  replay) is not recorded as closed.
- `milestones.md` still says "All milestone exits below remain open". No
  document maps a scenario to a milestone.

**Effect:** the shell grows faster than the model below it. The handoff shows
this: Refine Appearance is blocked because the sketch prism declines curved
profile boundaries. A UI scenario found a modelling gap that R5 owns.

### 2.2 The R0 exit was skipped, so there is no measure of progress

`verification.md` requires a checked-in, machine-readable behavior ledger with
a stable ID, source location, target module, milestone, and test for each row.
CI must fail on an unclassified row. This ledger does not exist. The only
ledgers are the BDD source inventory (file hashes) and a prose work ledger.

**Effect:** "how complete is the port" has no answer. The 24 ledger rows mix
whole areas ("Other `features/sketch`") with single scenarios.

### 2.3 The benchmark moves faster than the agent can verify it

- The scenario source changed in 23 of 24 application files after the
  checkpoint. The checkpoint was not advanced.
- Each change makes earlier results "historical". Four rows are "Needs
  rerun". Much of the day went to wording updates of step bindings.
- The agent runs against the live directory, not a pinned revision.

**Effect:** the agent chases a moving target. Verified work decays without a
code change.

### 2.4 The evidence rules are not obeyed, and they cost too much

- The workflow says reports must be in durable storage. The ledger points to
  `/tmp/obk-bdd-qat-…` directories. A reboot deletes the proof.
- Each small step adds 15 to 20 lines of prose to the ledger. The ledger has
  57 evidence sections, almost all from one day.
- Status, history, and evidence are in one file. The agent must read 94 kB to
  find the next task.

**Effect:** the agent spends tokens to write and read prose that a tool can
generate. The proof is not auditable.

### 2.5 The agent has no definition of a work unit

The plan has no rule for the size of one iteration. The observed pattern is:

1. Run a feature; get "N stopped at undefined steps".
2. Add bindings until the steps are defined.
3. Discover the missing behavior behind the bindings.
4. Implement part of it; commit; write evidence; go to the next scenario.

A binding that exists is counted in the same test run as a behavior that
works. "62 of 71 stopped at undefined steps" hides the true size of the work.
The first failure also stops the scenario, so later gaps stay hidden.

### 2.6 `AGENTS.md` contained rules from other projects

Parts of `AGENTS.md` came from the `ai-memory` and Martensite repositories
and did not apply here. They were removed on 2026-11-13 (431 → 336 lines):

- "`ai-memory-hooks` sanitizer", capture exclusions, "`purge-project`,
  `reset`, `restore`".
- "`.github/workflows/martensite.yml`" (the real files are `ci.yml`,
  `build.yml`, `nightly.yml`), `cargo xtask pr`, and the `install-*.sh`
  installers. These do not exist in this repository.
- The unsafe-exception rules with issues `#80` and `#88` (the workspace
  forbids `unsafe`), and the package-version rule with `#256`.
- "MCP tools, config keys, a new agent harness" in the versioning rule.
- The "patch line / standing release candidate branch" model.
- The `humanizer` rule and the rule for tests that build Git repositories.
- The empty "Cross-cutting invariants (do not violate)" section.

Still open: "Repository layout" lists two directories only, and the file
conflicts with `plan.md`:

| Topic | `plan.md` | `AGENTS.md` | Actual |
| --- | --- | --- | --- |
| Work branch | `rust-port`, single branch | `rust`, one PR for each integration | `feat/rust-consolidated`, local only |
| Feature docs | `architecture/rust-port/implementation-*.md` | `docs/features/<name>/` | 200 files in the first, 1 entry in the second |
| Gate | `make gate` | `make gate` (corrected) | not recorded for each commit |
| Tracking | epic #3563 in `Oblikovati/Oblikovati` | "issues on this repo" | 2 % of commits refer to an issue |

**Effect:** an agent that reads conflicting rules selects one at random or
ignores all. Each irrelevant line also costs context in each session.

### 2.7 The work is not safe

- 981 commits exist only on one disk.
- 25 merged local branches and 2 old worktrees remain.
- The agent session can stop without a handoff (this occurred today: a
  `codex` session ended with uncommitted edits in seven files).

### 2.8 Quality gates are stated but not run at the unit of work

- 20 files break the 500-line rule, 5 of them in `src/`.
- The gate requires coverage above 80 % and duplication below 3 %, for each
  crate. No receipt is recorded for the current branch.
- Commits have a subject line only. The rule asks for the what and the why.
- Pixel checks in native tests were written against a green fixture color.
  The handoff reports that the color was an assumption, not a requirement.
  This is a test with a bad premise.

### 2.9 What is good and must stay

- Clear ownership: API contract, Martensite kernel, application.
- "Source presence, unit test, and user behavior are separate states."
- Hash-based scenario inventory with before/after check.
- Native-window BDD with real mouse and keyboard, not API-only proof.
- Kernel and scenario defect records in separate directories.
- The rule that an undefined, skipped, or filtered case is not a pass.

## 3. How the agent can be more effective

1. **One instruction source.** `AGENTS.md` holds rules only, less than 200
   lines, all applicable. It points to one workflow file.
2. **One queue.** A generated file gives the next task. The agent does not
   select work from prose.
3. **Pinned inputs.** The agent works against a scenario commit, not a live
   directory. Scenario updates are a separate, scheduled task.
4. **Size before work.** A dry run that does not stop at the first failure
   gives the full list of gaps before the agent writes code.
5. **Vertical slices.** One user capability goes through model, session,
   automation, GUI, and scenario in one unit. No binding-only commits.
6. **Stop rules.** A time or attempt limit on each unit. When the limit is
   reached, the agent records a blocker and takes the next unit.
7. **Generated evidence.** A tool writes one JSON receipt for each run. The
   agent writes no evidence prose.
8. **Short context.** Status file below 150 lines. History in archives that
   the agent does not read by default.
9. **Checkpoint each unit.** Commit, push, and update the handoff at the end
   of each unit, not at the end of the session.
10. **Fast inner loop, full outer loop.** Owner and model tests in seconds
    during the work; native run for the feature at the end of the unit; full
    gate once for each batch before a push.

## 4. The iteration loop

One iteration delivers one **capability**: a group of scenarios in one feature
file that need the same behavior (for example "Quick Access: Refine
Appearance"). Target size: one to eight scenarios, one to three commits.

### Step 0 — Start the session (5 minutes maximum)

1. Read `STATUS.md` (new, see action A4) and the pending handoff. Read
   nothing else.
2. Run `git status`. If there are uncommitted changes from an earlier
   session, complete or revert them first. Do not start new work on top.
3. Confirm the pinned scenario revision. Do not triage scenario changes now.

### Step 1 — Select

1. Take the first row of the queue with state `ready`. The queue order is:
   milestone order first, then scenario priority in the milestone.
2. Confirm that a GitHub issue exists for the capability. Create it if not.
3. Confirm that the dependencies of the row are `verified`. If not, the
   dependency is the task.

### Step 2 — Size (no production code)

1. Run the feature with the dry-run mode (action A6). It reports each
   scenario as: `passes`, `unbound step`, `bound but fails`, `blocked by
   fixture`.
2. Read the Go reference for the behavior. Record the file paths in the issue.
3. Classify each gap by owner:
   - **application** → continue;
   - **contract** → API-first change in `Oblikovati.API`, in one batch;
   - **kernel** → record in `kernel-defects/`, mark the scenarios `blocked`,
     and stop work on them;
   - **scenario defect** → record in `scenario-defects/`, do not weaken the
     check.
4. Write the plan in the issue: scenarios in scope, layers to change, tests
   to add. If the plan needs more than eight scenarios or three layers of
   new design, divide it.

### Step 3 — Red

1. Write the failing test at the lowest layer that owns the behavior (model
   or session), with an independent expected value.
2. Bind the scenario steps. A step binding must call the real native window
   path. Run the scenario and confirm that it fails for the correct reason.
3. Check the premise of each assertion against the Go application or the
   scenario text. Do not assert a value that only the current fixture gives.

### Step 4 — Green, from the bottom up

1. Model and recipe, with persistence.
2. Session command, transaction, undo/redo.
3. Automation method (contract first if the surface is new).
4. GUI command, ribbon or browser exposure.
5. After each layer, run only that crate's tests.

Rules: no fallback, no stub that returns success, no change to a tolerance or
an assertion to pass. A missing kernel capability is a named decline with a
diagnostic.

### Step 5 — Verify

1. Run the full feature file in the native window under Xvfb, against the
   pinned scenario revision. Save the receipt.
2. Look at a screenshot for each new visible behavior. Check the model state
   with a query, not only pixels.
3. Run the tests of each crate that depends on the changed crates.
4. Run format and clippy.

### Step 6 — Record and checkpoint

1. Commit with a subject, a body that gives the reason, and `Refs #issue`.
2. Let the tool update the queue and `STATUS.md` from the receipt.
3. Push the branch.
4. Close the test windows. Remove temporary report directories that the
   receipt does not refer to.

### Step 7 — Stop or continue

| Condition | Action |
| --- | --- |
| All scenarios in scope pass on the full feature run | Mark `verified`; go to Step 1. |
| Two fix attempts fail on the same cause | Stop. Write the hypothesis and the evidence in the issue. Use the debugging procedure, not a third variation. |
| The unit exceeds 90 minutes or three commits without a new passing scenario | Stop. Record `blocked` with the cause. Go to Step 1. |
| The gap is in the kernel or the contract | Record it and go to Step 1. Do not build an application workaround. |
| The scenario source hash changed during the run | Discard the run. Do not triage now; record it for the scheduled sync. |

### Batch gate (each 5 to 10 units, and before each PR)

1. `make gate` on a real or Xvfb display. Record the receipt.
2. Coverage and duplication for each crate.
3. Full area run for the area that changed, to detect regressions.
4. Merge upstream, then open one PR for the batch.

### Scenario sync (scheduled, not continuous)

1. One time for each week, or on your request, move the pin to a new scenario
   commit.
2. The inventory tool lists the changed files. Mark their rows `needs rerun`.
3. Rerun them in one batch. Fix wording-only binding changes in one commit.

## 5. Ordered actions

### P0 — Protect the work (today)

| ID | Action | Done when |
| --- | --- | --- |
| A1 | Complete or revert the uncommitted appearance change. Verify the framebuffer colors against the resolved appearance, not the green constant. | `git status` shows only your `README.md` and `AGENTS.md`. |
| A2 | Push `feat/rust-consolidated` to the remote. | `git rev-list --count origin/feat/rust-consolidated..HEAD` is 0. |
| A3 | Delete the 25 merged local branches and the two old worktrees, after a check that each is an ancestor of the branch. | `git branch` shows the work branch and `go-legacy` only. |

### P1 — Give the agent one source of truth (one session)

| ID | Action | Done when |
| --- | --- | --- |
| A4 | Create `architecture/rust-port/STATUS.md`: current milestone, the next five queue rows, open blockers, the scenario pin. Maximum 150 lines. Generated where possible. | The agent can start from this file only. |
| A5 | Complete `AGENTS.md`. The rules from other projects are removed (see 2.6). Remaining: write one branch policy, one feature-doc location, and the repository layout; confirm `cargo deny`, `cargo audit`, and `cargo xtask release`. Align `plan.md` "Branch and work policy" with it. | No rule refers to a file, issue, or tool that does not exist in this repository. |
| A6 | Add a dry-run mode to the BDD runner that continues after a failed step and classifies each scenario (see Step 2). | One command gives the gap list for a feature. |
| A7 | Pin MCAD-Scenarios to a commit. Record the commit in `STATUS.md`. `OBK_BDD_ROOT` points to a checkout at that commit. | A run records the scenario commit, not only file hashes. |

### P2 — Make progress measurable (two to three sessions)

| ID | Action | Done when |
| --- | --- | --- |
| A8 | Build the machine-readable ledger that R0 requires: one row for each scenario (2719 rows), with feature, milestone, owner layer, state, last receipt. Generate it from the feature files and the receipts. | A script prints verified / total for each area and each milestone. |
| A9 | Map each feature file to a milestone (R3 to R8). Reorder the queue: R3 exit first, then R4, then R5 families with their scenarios, then R7 shell detail. | `milestones.md` shows a state for each exit, with a link to evidence. |
| A10 | Replace the prose evidence with JSON receipts in a directory in the repository (or an artifact store that you select). Move the current 1395-line ledger to an archive file. | The ledger table is below 150 lines; no evidence points to `/tmp`. |
| A11 | Close R0, R1, R1V, R2, and R3 in a formal way: run the exit checks, record the receipts, or list the gaps as issues. | Each of the five milestones is `closed` or has a named gap list. |

### P3 — Restore the quality bar (continuous)

| ID | Action | Done when |
| --- | --- | --- |
| A12 | Divide the 20 files above 500 lines. Add the line limit to `cargo xtask lint` so that it fails. | The lint fails on a new large file. |
| A13 | Run `make gate` and record coverage and duplication for each crate. Create issues for crates below the limits. | A receipt exists for the current branch. |
| A14 | Audit the native pixel assertions. Each one must state the expected color source (theme, appearance, material) and must fail when the behavior is removed. | No assertion depends on a fixture constant without a stated source. |
| A15 | Require `Refs #issue` and a body in each commit by a commit-message hook. | The hook rejects a subject-only commit. |

### P4 — Decisions that you must make

**Decided 2026-11-13:** Use milestone order with BDD acceptance, sync the
scenario pin weekly, store JSON receipts in the repository, use one working
branch with batch PRs to `rust`, and track application work in Oblikovati
issues and validated kernel gaps in Martensite issues for the kernel agent.

1. **Order of work.** Milestone order (model depth first) or benchmark order
   (application shell first)? Recommendation: milestone order, with the
   scenarios as the acceptance test of each milestone. The shell work that
   exists stays; new ribbon detail waits for R3 and the first R5 families.
2. **Benchmark change rate.** Is a pin with a sync one time for each week
   acceptable? Without a pin, verified work decays each day.
3. **Evidence storage.** In the repository (small JSON), or an external
   artifact store?
4. **Branch model.** One long branch with batch PRs to `rust`, or one PR for
   each capability? Recommendation: one branch, one PR for each batch of 5 to
   10 units, pushed after each unit.
5. **Kernel gaps.** Who does the Martensite work when a scenario is blocked:
   the same agent in a separate session, or a separate queue?

## 6. Expected result

- The agent starts a session with one short file and one queue row.
- Each unit ends with passing scenarios, a pushed commit, and a generated
  receipt, or with a recorded blocker.
- You can read the port state as a number for each milestone and each area.
- Scenario changes cost one scheduled batch, not daily rework.
