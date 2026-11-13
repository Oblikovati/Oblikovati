# BDD work ledger

This ledger records active work and its evidence. Follow [the BDD workflow](bdd-workflow.md)
before changing a status. The source inventory is a discovery checkpoint, not
a pass count. The initial inventory records the current files; it does not
claim that each file has been individually triaged.

| Priority | Feature and scope | State | Evidence and next action |
| --- | --- | --- | --- |
| 1 | `features/application` | In progress | The native application runner exists. A complete, current area result has not been recorded. Run and retain the full area report; triage failures against the current source. |
| 2 | `features/sketch/pattern-rectangular/scenario.feature` | In progress | Rectangular patterns can be created and edited in the native UI, but directions are fixed to the sketch axes. Implement line-reference selection and associativity, then run the affected native scenarios. |
| 2 | Other `features/sketch` | In progress | The native sketch gate runs only selected folders. Inventory and run the remaining folders before reporting total coverage. |
| 3 | `features/solid-model` | In progress | Implementations exist, but this area has not been assessed against the current complete source. Assess each feature after the application and sketch priorities; no total execution result is recorded. |

## Evidence records

There is no current full-suite native report in this ledger. The most recent
default report seen during setup contains one passing application scenario;
it is a filtered run and does not establish an area total. Existing targeted
runs remain useful development evidence but need their source and application
revisions checked before they support a current pass claim.
