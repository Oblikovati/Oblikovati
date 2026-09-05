# Prompt — picking up milestone M48 (Kernel Ground-Rules Remediation)

Paste everything below the line into a fresh agent session in `/Oblikovati`.

---

You are working the GitHub milestone **"M48: Kernel Ground-Rules Remediation (2026-08 kernel-shape audit)"**
in `Oblikovati/Oblikovati`. Read `CLAUDE.md` first — the section **"Kernel & host: ground rules for
extension and refactoring"** is the law for every change in this milestone; the rest of `CLAUDE.md`
(code style, tests, git, PR gates) still applies.

## What the milestone is

Eight **Feature** issues, one per design-problem class, each holding **Task** sub-issues, one per code
occurrence (`file:line`). Tasks carry a phase:

- `refactor-only` label = **phase 1**: behaviour-preserving reorganization (delete a duplicate, move a
  type switch into `kernel/geom`, add an `archguard` rule, split a file, annotate a tolerance, impose a
  total order). Byte-identical geometry before and after is the contract.
- no label = **phase 2**: an algorithmic change (fold a recognizer into the general pipeline, route a
  classifier through `kernel/predicates`, make `Validate` a post-condition, replace mesh-based
  containment with the exact B-rep).

Feature order is the work order: **C8 → C2 → C7 → C4 → C6 → C1 → C5 → C3**. Inside a Feature, tasks are
listed top-to-bottom in work order; phase-1 tasks come first. Task groups (`[M48/Cn group k/m]`) exist
only because GitHub caps sub-issues at 100 — treat a group as a slice of its Feature.

## How to pick the next task

1. `gh issue list --milestone "M48: Kernel Ground-Rules Remediation (2026-08 kernel-shape audit)" --state open --limit 500 --json number,title,labels`
   and take the **lowest-numbered open Task** (title starts with `[Cn]`) that carries `refactor-only`.
   Only when no `refactor-only` task is open anywhere in the milestone do you take the lowest-numbered
   phase-2 task. Never skip ahead to a "more interesting" task.
2. Read the Task: **Occurrence**, **Problem**, **Rule violated**, **Acceptance criteria**, and any
   "Same site is also tracked from another class" line. If several tasks name the same `file:line`,
   take them together in one PR and close all of them.
3. Check the site is still as described (`sed -n`, `grep`). If it moved, update the Task body with the
   new location before you start. If it no longer exists, say so in a comment, close the Task as
   "not planned", and take the next one.
4. Comment `Taking this` on the Task and open a branch `m48/<issue-number>-<slug>` from `develop`.

## How to do a task

- **Phase 1 is a refactor, not a fix.** Lock the current behaviour first: the existing corpus/goldens
  for that path must pass before and after with no golden updates. If you find a bug on the way,
  file it (Bug type, milestone M48, `Related: #<task>`) and leave the behaviour as it was.
- **Phase 2 changes behaviour through the general path only.** The rule is in the Task body; the
  target end state is the last sentence of **Problem**. Add the failing input to the operation's corpus
  as a test that goes through the general implementation. Adding a new recognizer, tolerance constant,
  fallback, retry, `if <special configuration>` branch, or type assertion to close a task is a failure
  of the task — the net delta of those four counts must be ≤ 0 in the PR (rule: *Every kernel PR reports
  the net change…*).
- **Delete what you replace.** A task that introduces the general path is closed only when the special
  case it replaces is gone (rule: *A generalization is complete only when the special cases it replaces
  are deleted*).
- **Validity.** Every body a changed operation returns is validated (`ops.Validate`, all levels that
  exist) in the test; an invalid body is an error, never a return.
- **Oracle.** Where the Task names an oracle (OCCT parity, per-face `sprops`, analytic volume), gate
  per-face, never whole-body only (rule: *Result gates are per-face*).
- **Tolerances.** Any comparison you touch reads `geom.Resolution` or carries `// tol:<kind>`; classify
  the operands' origin before choosing `Weld()`/`Sew()`/`Plane()` (rule: *Classify a comparison by the
  origin of its operands*).
- **Naming.** Any entity your change generates gets parent-derived lineage under the feature's unique
  tag, and any lookup you touch goes through the guarded resolver (`FacesByKey`/`EdgesByKey` + the
  collision error), never `FindXByKey`.
- **Do not touch `kernel/meshbool` exactness, `reconstructionCutover`, or the ADR-0047 radial-edge sew**
  except in a task that names them.

## Definition of done for one task (all four, then the PR)

1. The occurrence at the named location is gone, and `grep` proves it.
2. The corpus/regression test for that path passes through the general implementation; for phase 1,
   goldens are byte-identical.
3. Full local suite + lint + `go test ./archguard/...` green; coverage >80% on touched packages,
   duplication <3%; the four net-delta counts are ≤ 0 and stated in the PR body.
4. Live test through `Oblikovati.AddIns.MCPBridge` with a screenshot for any task that changes rendered
   geometry (tessellation, boolean, fillet).

Then: one PR per task (or per same-site task set), body = WHAT and WHY, `Closes #<task>` per task,
mention the parent Feature. Merge when CI is green, `git fetch && git pull` on `develop`, delete the
branch, tick the task in the Feature's checklist, and go back to step 1.

## When a task is bigger than it looks

Capabilities are layered: the next blocker is hidden until the first one is fixed (rule: *Size a
cluster by driving one representative case to a valid solid first*). If a phase-2 task uncovers a
prerequisite (e.g. the general SSI engine, `geom.Surface` interface deepening, `kernel/topo/provenance`
seam), do NOT special-case around it: comment on the Task with the prerequisite, open a new Task for it
(milestone M48, same Feature, `Blocks: #<task>`), take the prerequisite first.

## Reporting

At the end of every session post one comment on the **Feature** you worked: tasks closed (numbers),
tasks blocked (numbers + blocker), net-delta counts for the session, and the next task number. Keep
`architecture/decisions/` current: a task that changes an ADR's status or supersedes a decision ships
the ADR edit in the same PR (rule: *An ADR is superseded by a new ADR, never edited in place*).
