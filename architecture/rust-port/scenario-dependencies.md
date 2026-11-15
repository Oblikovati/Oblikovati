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
It inventories the full source checkout, including Git metadata, and verifies
the source on both sides of the run. Nested user namespaces are disabled.
Read-only mounts and before-and-after inventories cannot prevent host-side
changes during a run. The host kernel, devices, clock, random inputs and
sandbox bootstrap dependencies also remain outside this inventory. Its observation always
sets `reusable` to false: an independent proof checker, rebuilt runner identity,
and complete host and device input policy are still needed. The ordinary native
receipt path does not use this helper. See
`receipts/2026-11-15-sealed-runner-probe/` for the live probe.

## Native build provenance

Native capture receipts remain available as historical evidence, and their
contents and artifact hashes are still checked. They do not count as verified
BDD scenarios: the current capture path does not prove the inputs used to
build the runner. A clean Git commit alone does not bound Cargo settings,
compiler wrappers, ignored files, or external build inputs. The ledger reports
this limitation as a stale reason and retains the historical receipt.

Issue #3978 tracks a bounded build and independently checked provenance.
The experimental sealed runner bounds parts of execution; it does not prove
how the runner was built and does not remove this limitation.

## Committed build source experiment

On POSIX hosts, `tools/bdd/committed_build_source.py` exports a Git commit to a new directory
outside the checkout. It checks every archive file against its tracked Git blob,
rejects links and submodules, and refuses missing, extra, duplicate or altered
files. Ignored, untracked and dirty worktree files are not copied. It uses a
restricted Git environment. For example:

```sh
python3 -B tools/bdd/committed_build_source.py "$PWD" FULL_COMMIT_SHA \
  /absolute/path/to/new-source > /absolute/path/to/source-inventory.json
```

The inventory includes SHA-256 hashes and always reports `reusable: false`.
The output directory is writable; changes after export are not prevented or
checked by this command. The command does not bound Cargo dependencies,
compiler and linker binaries, build scripts, environment, system libraries,
or host changes during a build. It is a source preparation experiment, not a
build or receipt proof. It is not connected to the native capture path. #3978
remains open until those inputs and the built runner are independently checked.

## Offline sandbox build experiment

`tools/bdd/sandbox_build_probe.py` runs a fixed Linux x86-64 head-test build
with an empty target, an empty Cargo home, no inherited environment, and an
isolated network namespace. It mounts source, vendored packages, the supplied
Rust toolchain and Cargo configuration read-only. It inventories those inputs
before and after the build and refuses a change. It finds the test executable
from Cargo's artifact messages and records its hash. Both output directories
must be empty and all input and output roots must be disjoint.

The configuration directory must contain `config.toml` with Cargo source
replacement configured to use `/vendor`. Each input root and the Bubblewrap
executable and its parent must support the no-atime checks. For example, use a
task-owned Bubblewrap copy.
Invoke the probe with seven absolute paths:

```sh
python3 -B tools/bdd/sandbox_build_probe.py SOURCE VENDOR TOOLCHAIN CONFIG \
  EMPTY_TARGET EMPTY_LOGS BWRAP > /absolute/path/to/build-observation.json
```

To compare the source snapshot with a Git commit before starting Cargo, add
`--repository /absolute/path/to/repository --revision FULL_COMMIT_SHA` to that
command. Both options are required together. The check creates a separate
verified export, then compares the mounted source's complete file and directory
set, file contents, file permissions and link counts. It reports the selected commit only
when the comparison succeeds. Without those options, it does not perform this
comparison.

The tool does not prove that the vendor directory matches trusted registry
content. It uses the host `/usr`, GCC, kernel, devices and clock. The read-only
mounts do not stop host-side writes during the build, and comparing snapshots
before and after cannot detect every transient change. The input inventory
hashes identify observations, not independently checked build provenance.
The result always says `reusable: false`; the normal receipt path does not use
this probe. Do not use it to mark a BDD scenario verified. Track #3978.
