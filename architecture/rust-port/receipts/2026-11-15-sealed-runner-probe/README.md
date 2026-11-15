# Sealed runner probe

On 2026-11-15 the sealed execution helper ran the pinned document-opening
scenario in Linux Bubblewrap. All nine steps passed. The helper verified the
source before and after, hashed the runner and sandbox executable, and checked
that all 556 runtime entries were unchanged across the run. The screenshot,
Cucumber report, source report, native log and runtime observation are retained.

The helper clears the environment, isolates the network and process namespace,
and mounts the runtime and scenario source read-only. The project checkout and
`/proc` are not mounted. This probe used the same copied runtime root and runner
binary identified in `receipts/2026-11-15-dependency-sandbox-probe/`.

This is not a reusable receipt. The observation marks runtime coverage
`unbounded` and `reusable` false. It does not yet establish an independently
verified execution proof, a rebuilt runner identity for a changed application,
or all host kernel and device inputs. Input was injected into the frame loop.
Track completion in #3974.
