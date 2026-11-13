# Historical BDD work ledger, part 4

This is an archived historical record, not current acceptance evidence. Old status labels and temporary report paths below are retained only for provenance. Use the [current ledger](../bdd-work-ledger.md) for current status.

## Evidence records

### Updated application source assessment, 2026-11-13

- The live inventory has 24 application feature files and 746 scenario
  declarations. Twenty-three files differ from the discovery baseline.
  These are declaration counts, not native test results.
- Captured all 24 application feature files in
  `/tmp/obk-application-snapshot-20261113`, verifying the live source before
  and after the copy. The manifest is `application.sha256` in that folder.
  A full native application run was started with no feature or scenario filter;
  its report is `/tmp/obk-application-full-20261113`. It completed with 105
  of 718 executed scenarios passing across 23 parsed features; the runner
  reported one parsing error. Home recent-documents did not parse because
  the source contains lowercase malformed lines such as `and opens Selected`;
  its scenarios are excluded from that run. Check the live source against the
  manifest before treating the finished report as current. Viewport
  orientation and viewport zoom changed after the snapshot was captured;
  their results will require reruns on the new source.
- A separate File menu run on the same snapshot passed all eight scenarios
  and 59 steps. File SHA-256:
  `fd33924caea15cf97ba75ad3600c7244cf17592b6bdf26bdc9882f7cae71fb91`.
  Application: `685571cca` plus the quick access Projects changes in this
  work, with unrelated README and AGENTS files present. Linux, Xvfb at
  1600 × 1000 and Lavapipe. Local report:
  `/tmp/obk-application-file-menu-current-20261113`.
- Projects is now a default quick access command that opens the existing
  Projects dialog. A native test loaded a project, opened the dialog from a
  part, checked its name, and returned to the same part. The focused toolbar
  choices scenario progressed past Projects and failed at the missing Update
  choice. GUI library: 358 passed; File dialog: four passed; GUI/head Clippy
  passed. The full toolbar feature needs a new run after the layout change.
- A full document-creation run after updating common action bindings passed
  eight of 39 scenarios and 145 of 176 executed steps. All 31 failed scenarios
  stopped at undefined steps, so later behavior was not assessed. Source
  SHA-256 `e9a93d4c096090667952b8080a0cad01c48f480022a4b8ac72ae210607ca138d`
  matched the live file after the run. Application: `14f93537f` plus these
  bindings; Linux, Xvfb at 1600 × 1000 and Lavapipe. Local report:
  `/tmp/obk-document-creation-reassessment-20261113`.
- A full document-opening run after updating action wording passed 16 of 27
  scenarios and 184 of 195 executed steps. The other 11 scenarios stopped at
  undefined steps; no later behavior is claimed. Source SHA-256
  `d7965f270ecc21d99f2ed9a0837f17a1996203fccf4d4d1966e7876ebb01464c`
  matched the live file after the run. Application: `5b88dbb7d` plus these
  test bindings; unrelated README and AGENTS files were present. Linux, Xvfb
  at 1600 × 1000 and Lavapipe. Local report:
  `/tmp/obk-document-opening-reassessment-20261113`.
- The focused `Inspect a part file before opening it` scenario passed all 12
  steps after its saved-part fixture returned to the original active part.
  The final assertion now checks that exact part after cancel. Source:
  `d7965f270ecc21d99f2ed9a0837f17a1996203fccf4d4d1966e7876ebb01464c`;
  application `f6e852fb1` plus this binding and fixture correction. Linux,
  Xvfb at 1600 × 1000, Lavapipe. Local report:
  `/tmp/obk-open-inspect-part-20261113`. This focused result does not update
  the earlier full-feature pass count.
- Three further focused document-opening scenarios passed on the same source:
  reverse size sorting (seven steps), inspect design views (10 steps), and
  navigate to a parent folder and back (13 steps). Application `17ed5d68b`
  plus the updated bindings; Linux, Xvfb at 1600 × 1000, Lavapipe. Local
  reports begin `/tmp/obk-open-focus-` and end `-20261113`. The complete
  feature was not rerun with these bindings.

### Model browser source reassessment, 2026-11-13

- Updated existing native actions for exact current-source wording and added
  matching block fixtures. A full run passed 29 of 100 scenarios and 523 of
  594 executed steps; 62 scenarios stopped at undefined steps and nine at
  failed checks. No ambiguous step was reported. Source SHA-256
  `603382ec4009d73a74e960c5981d255c4ca9ac5489b94fcf57193c7b09458b2e`
  matched the live file after the run. Application: `e94784d32` plus the
  initial wording and fixture changes. Linux, Xvfb at 1600 × 1000, Lavapipe.
  Local report: `/tmp/obk-model-browser-reassessment-20261113`.
- Later focused runs passed filtering and clearing browser search (11 steps),
  canceling a sketch line-style change (12 steps), declining an extra Find
  criterion, canceling adaptive feature options, and inspecting a nested sketch
  (10 steps). The sketch check reads selected identity and an uncovered pixel;
  its context menu covers the earlier pixel sample. Those fixes correct test
  actions and fixtures; they do not change the recorded full-run count.
  The source's `collapses All Children` wording remains unbound pending a
  scenario correction or clarification.
