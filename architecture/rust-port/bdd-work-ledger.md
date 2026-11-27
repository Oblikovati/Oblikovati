# BDD work ledger

The machine-readable [behavior ledger](behavior-ledger.json) is the current
scenario inventory. A scenario is verified only when its current native receipt
passes the checks in [behavior receipts](behavior-receipts.md). Historical
claims do not change that state. See [status](STATUS.md) for the active queue.

To inspect the current source and feature backlog:

```sh
python3 -B tools/bdd/behavior_cli.py report
python3 -B tools/bdd/behavior_cli.py backlog
```

The prior prose ledger is preserved for provenance in four parts:

1. [Historical ledger, part 1](archive/bdd-work-ledger-2026-11-13-1.md)
2. [Historical ledger, part 2](archive/bdd-work-ledger-2026-11-13-2.md)
3. [Historical ledger, part 3](archive/bdd-work-ledger-2026-11-13-3.md)
4. [Historical ledger, part 4](archive/bdd-work-ledger-2026-11-13-4.md)

The archive includes old status labels and temporary output paths. Those paths
are not retained acceptance evidence. Rerun a case and save a current receipt
before reporting it as verified. Follow the [BDD workflow](bdd-workflow.md) for
source changes, triage, and new evidence.
