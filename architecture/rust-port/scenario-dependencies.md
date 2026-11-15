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
Git commands use a restricted environment and ignore replacement objects. The
clean-worktree check also refuses index entries marked assume-unchanged or
skip-worktree. Revisions containing Git submodules are refused because the map
does not inventory their contents, including uninitialized or ignored changes.
Ignored files and host inputs remain outside this check. Unknown
schemas, scenario changes, missing records and divergent histories are refused. Keep output outside the worktree to avoid making the map dirty.

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

The experimental `sealed_build_chain.py` connects a committed build probe to a
sealed execution in one process. It checks the observed runner and Bubblewrap
hashes, copies the runner into a Linux memfd, seals it against writes and resizing,
and supplies it to Bubblewrap using `--ro-bind-data`. The execution observation
must match the build observation. This prevents a later path replacement from
selecting different runner bytes, assuming the kernel and Bubblewrap behave as
expected. It always returns `reusable: false` and is not connected to native
receipt verification. Host tools and system files, dependency manifests,
kernel and device inputs, trusted sandbox identity, concurrent modifications,
and the rest of the runtime remain unproven. The optional vendor checks do not
verify generated Git manifests. `tools/bdd/test_pinned_runner.py` exercises a
real Rust test binary and path replacement; it does not run an Oblikovati UI
scenario. A real pinned Oblikovati build and native scenario also passed through
the CLI; see the [sealed build chain diagnostic](receipts/2026-11-15-sealed-build-chain-diagnostic/README.md).
That run remains non-reusable and does not close these gaps.

Run the linked diagnostic with explicit, disjoint inputs. The source must be a
verified export of the selected application commit. The vendor and Cargo
configuration must be prepared as described under the offline build experiment.
The target, build logs, and reports directories must be empty. The runtime must
contain the slots required by the sealed runner, Xvfb, and its runtime files.
The command needs Linux memfd sealing and Bubblewrap `--ro-bind-data` support.
For example:

```sh
python3 -B tools/bdd/sealed_build_cli.py \
  --source /absolute/source --vendor /absolute/vendor \
  --toolchain /absolute/toolchain --config /absolute/config \
  --target /absolute/empty-target --build-logs /absolute/empty-build-logs \
  --bwrap /absolute/bwrap --repository /absolute/repository --revision FULL_SHA \
  --runtime /absolute/runtime --checkout /absolute/pinned-scenarios \
  --reports /absolute/empty-reports --run-log /absolute/new-native.log \
  --area application --feature document-opening --scenario 'Open a part' \
  > /absolute/diagnostic-observation.json
```

Add `--registry-archives /absolute/cache` to check registry packages and one
`--git-repo SOURCE /absolute/repository` for each locked Git source to run the
partial Git vendor checks. Omitting them leaves those checks unperformed. The
scenario string is a filter; inspect the output report for the executed cases.
The CLI writes an observation only. It does not publish or verify a receipt.

## Native build provenance

Native capture receipts remain available as historical evidence, and their
contents and artifact hashes are still checked. They do not count as verified
BDD scenarios: the current capture path does not prove the inputs used to
build the runner. A clean Git commit alone does not bound Cargo settings,
compiler wrappers, ignored files, or external build inputs. The ledger reports
this limitation as a stale reason and retains the historical receipt.

Native capture now exports the selected commit into a temporary source tree and
uses a fresh, temporary Cargo target. This excludes ignored checkout files from
the initial source tree and prevents reuse of a caller's target. It makes each
capture rebuild and removes the temporary source and target after the run. The
export is not sealed during compilation. Cargo settings, compiler and linker
selection, build scripts, external files, dependencies, system libraries and
concurrent writes remain unbounded. Runtime assets or add-ins that exist only in
the original checkout or beside an old executable are not copied. A capture
can fail when it depends on them; it must not be treated as verified evidence.

The capture command selects a fixed set of environment variables and uses a
fresh Cargo home. It copies every locked crates.io archive after checking its
SHA-256 against Cargo.lock. It copies the registry index into the temporary home;
the index contents are not authenticated. It stages shallow Git databases for
each locked commit and its recursive gitlinks, then checks their objects with
Git fsck. Cargo creates fresh registry sources and Git checkouts when the
committed source settings select them. Staging excludes caller Git config and
refuses ambiguous commits, alternate object stores, and shallow source databases.

The project Cargo config is restricted to the current xtask alias and Linux
rpath setting. Other settings are refused until reviewed. All exported Cargo manifests
must not contain patch or replace tables, and their path fields must remain
inside the export. The config, manifests, and lockfile are checked again after each build. Display and renderer
variables are passed through and their values are not attested. Host tools,
Git SHA-1, the registry index, external files, dependency build scripts, and
concurrent changes remain unproven. Other processes can alter inputs during a
build or make a change and revert it between checks. Cargo offline requires all
locked dependencies in the local cache.

Native capture builds the inventory generator from the selected committed
source in a separate temporary offline target. The capture does not use the
caller-provided `OBK_SCENARIO_LEDGER_TOOL`; other ledger commands retain that
option. Snapshot and application Git identity checks discard caller Git
environment overrides. The inventory generator and Git checks still depend on
the host tools, local repository configuration, the registry index, and inputs
that may change during execution. They are not independent attestation.

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

The build probe requires Bubblewrap support for `--disable-userns` and disables
nested user namespaces. Without that restriction, a
build process on this Linux host could create a user and mount namespace, mount a
new tmpfs over its read-only `/source`, and compile different content while the
host source remained unchanged. The real Bubblewrap regression in
`tools/bdd/test_sandbox_namespace_integration.py` reproduces the substitution
without the restriction and verifies its refusal with the restriction. It skips
when the host cannot reproduce nested namespaces.

The build probe still mounts host `/usr` without inventorying it. A sandboxed
process can read `/usr/bin/gcc` and `/proc/uptime`; the compiler, system files,
kernel, devices, time, and concurrent host changes are not bounded. The Git
vendor check also leaves generated manifests unverified. The probe does not
feed the native capture command or independently verify the built executable.
It remains non-reusable and does not change the receipt verifier. A safe
end-to-end integration needs a separately checked immutable toolchain and
system runtime, complete dependency and generated-manifest verification,
controls for kernel and device inputs, a trusted sandbox and observation chain,
and an independently verified link between that build and the executed runner.

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

With `--repository` and `--revision`, add `--registry-archives /absolute/cache`
to compare crates.io vendor packages with the `.crate` files named by the locked
source. The check verifies each archive against its lockfile SHA-256, then checks
vendored paths, contents, executable bits and canonical checksum metadata. It
requires the explicit Cargo configuration to replace the locked sources with
`/vendor`, and refuses project Cargo source overrides. It accounts for Cargo's
VCS metadata exclusions and refuses unknown package sources or noncanonical
metadata. The archive cache must support no-atime reads. Cargo's
[checksum metadata](https://doc.rust-lang.org/cargo/reference/source-replacement.html)
is not itself a security mechanism; it is checked against the archive instead.
Git packages are listed as unverified. This check does not establish the whole
vendor directory as trusted.

To include the partial Git blob check in the same probe, also pass
`--git-repo SOURCE REPOSITORY` for every locked Git source. `SOURCE` is the
complete `git+...#commit` value from `Cargo.lock`; `REPOSITORY` is an absolute
local object repository. The probe refuses missing, duplicate or unexpected
source mappings and checks the vendor inventory before running Cargo. Its
`vendor_check` result contains both registry and Git observations. The Git
manifests remain unverified, and the combined result cannot authorize reuse.

The tool uses the host `/usr`, GCC, kernel, devices and clock. The read-only
mounts do not stop host-side writes during the build, and comparing snapshots
before and after cannot detect every transient change. The input inventory
hashes identify observations, not independently checked build provenance.
The result always says `reusable: false`; the normal receipt path does not use
this probe. Do not use it to mark a BDD scenario verified. Track #3978.

The separate `tools/bdd/git_vendor_check.py` compares Git-sourced vendor
files to blobs at the full locked commits. Give each locked Git source and a
local object repository with `--git-repo SOURCE REPOSITORY`. It checks the
package file namespace, file bytes, executable bits, and Cargo checksum
metadata. It reads Git objects, not checkout files. The check conservatively
requires tracked package files, apart from Cargo's VCS metadata exclusions.
A package that omits tracked files or inherits its version is refused.

Cargo 1.99.0 [normalizes Git vendor manifests](https://github.com/rust-lang/cargo/blob/5f94df4789f005f9a352888e8355ffc645b7ed0e/src/ops/cargo_vendor.rs#L454-L466)
after resolving workspace inheritance and filtering build scripts and targets.
The Git check now independently compares the parsed dependency, development and
build dependency, target, and feature declarations with the locked source
manifest. It also refuses a generated build script when the locked source has
none. It refuses source build scripts and declaration transformations it cannot
compare exactly, including workspace-inherited dependencies; this is a deliberate
compatibility limit. Boolean and integer TOML values are compared by type.

The check also validates the conventional inferred library target when the
locked source has no explicit `[lib]`, autodiscovery is enabled, and
`src/lib.rs` is tracked. Cargo's [pinned target normalizer](https://github.com/rust-lang/cargo/blob/5f94df4789f005f9a352888e8355ffc645b7ed0e/src/workspace/parser/targets.rs#L139-L180)
derives its name from the package name and uses that path. The generated library
must contain exactly those two fields. The check also requires normalized target
autodiscovery switches to be false, compares `package.links`, and refuses
`metabuild`. Explicit libraries and other layouts are declined until their
normalization can be checked.

These checks passed on the current 18 Git packages and 22,972 tracked vendor
files. They do not validate the entire generated manifest: package metadata,
explicit binary, example, test and bench targets, lints and other fields remain
unverified. A passing result is partial, does not independently bind a build,
and cannot authorize reuse. It reports the checked scope but keeps all 18 full manifests
in `git_manifests_unverified`, with `complete: false` and `reusable: false`.
The prior [observation](receipts/2026-11-15-git-vendor-check/observation.json)
and [regeneration experiment](receipts/2026-11-15-git-manifest-reproduction/README.md)
record earlier results.
