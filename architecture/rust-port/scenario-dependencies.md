# Scenario dependency maps

`tools/bdd/dependency_cli.py` generates one record for each pinned scenario.
Each record names the exact feature hash and runner. The shared map stores the
complete tracked Git tree, including file modes and symlink object identities.
It also records whether the worktree was clean. An optional runtime inventory
can be attached to one scenario; its files, directory entries, symlinks,
metadata and contents are checked without following symlinks. The map does not
infer independence from a scenario name, coverage, or a partial file trace.

```sh
python3 -B tools/bdd/dependency_cli.py generate --output /tmp/scenario-map.json
python3 -B tools/bdd/dependency_cli.py compare --map /tmp/scenario-map.json \
  --scenario scenario:FULL_ID

# Optionally record and compare one scenario runtime.
python3 -B tools/bdd/dependency_cli.py generate --output /tmp/scenario-map.json \
  --scenario scenario:FULL_ID --runtime-root /absolute/path/to/runtime
python3 -B tools/bdd/dependency_cli.py compare --map /tmp/scenario-map.json \
  --scenario scenario:FULL_ID --runtime-root /absolute/path/to/runtime
```

Comparison requires a clean recorded and current worktree, an ancestor commit,
the current pinned inventory, and an untampered complete recorded Git tree.
Added, removed, renamed, changed and mode-changed tracked inputs are reported.
The optional runtime inventory requires Linux no-follow and no-atime file
descriptors; unsupported platforms decline instead of claiming completeness.
Unknown schemas, scenario changes, missing records and divergent histories are
refused. Keep output outside the worktree to avoid making the map dirty.

The current map marks runtime coverage `unbounded`. It cannot authorize reuse
of a receipt even when its tracked input list has no changes. The normal ledger
keeps its existing conservative invalidation. A reliable selective policy also
needs a closed runtime namespace, a reproducible compiled runner identity, and
verified system, display, toolchain, environment and dynamic-input identities.
A Linux feasibility probe is recorded in
`receipts/2026-11-15-dependency-sandbox-probe/`; it is not that proof. Track
this work in #3974.

`sealed_execution.py` provides a Linux-only experimental runner. It clears the
inherited environment, starts Xvfb in a Bubblewrap namespace, hides the project
checkout, and mounts the runtime and pinned source read-only. It snapshots the
runtime and hashes the runner and sandbox executable before and after the run.
It also verifies the source on both sides of the run. Its observation always
sets `reusable` to false: an independent proof checker, rebuilt runner identity,
and complete host and device input policy are still needed. The ordinary native
receipt path does not use this helper. See
`receipts/2026-11-15-sealed-runner-probe/` for the live probe.
