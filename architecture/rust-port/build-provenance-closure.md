# Native BDD provenance: closure path

Issue #3978 prevents native receipts from counting as verified. This document
records the remaining chain and a proposed way to close it. It does not enable
receipt reuse or define a trusted builder.

## Current evidence and gaps

1. `capture_cli.py` exports the selected application commit, stages locked
   dependencies and builds in a fresh target. Its output is explicitly
   `historical_only`. The build still runs with host tools and libraries, an
   unauthenticated registry index, unbounded build-script reads and mutable
   files. Display and renderer inputs are inherited.
2. `sandbox_build_probe.py` can build an exported commit without network in a
   Bubblewrap namespace. It checks source and optional vendor inputs before the
   build and snapshots several mounts before and after it. It still mounts host
   `/usr`, `/proc` and `/dev`; the kernel, toolchain authority and external
   writes during a build are not bounded. Before-and-after equality cannot rule
   out a temporary substitution. Git vendor manifests remain partly checked.
3. `sealed_build_chain.py` binds a digest of the observed build output to sealed
   runner bytes in the experimental execution. This is useful identity binding,
   conditional on the observing process, kernel and sandbox. It does not prove
   the build inputs or the origin of the observations. The diagnostic CLI does
   not publish a native receipt or validate its scenario artifacts for reuse.
4. `behavior_evidence.py` checks report content, artifacts, source and commit
   identity, then unconditionally marks native receipts stale for #3978. No
   caller-provided receipt field changes this decision. The dependency map also
   refuses selective reuse while runtime coverage is unbounded (#3974).

## Proposed trust boundary

A verifier needs a trust policy outside the receipt. It must name the approved
builder identity, supported platform and immutable build/runtime image digests,
and the attestation format and version. A receipt cannot approve its own key,
image, policy or sandbox. The policy must also say which kernel, virtual device,
display and renderer inputs are trusted for the acceptance claim. This is a
trust assumption about the builder and its host, not a proof against a malicious
host administrator or a compromised signing key.

A possible implementation is a dedicated isolated builder with an authenticated
attestation key. It should use immutable, content-addressed input images and
prevent the job from writing them, modifying the builder, accessing an
uncontrolled network or using an unbounded host filesystem. The approved image
must include the Rust toolchain, linker, system files and build tools. Source,
lockfile and dependencies must be verified against approved identities before
the build. Build scripts may run only within the same bounded input namespace.
The runner must execute in an approved runtime with its display and renderer
inputs covered by the same policy. Snapshotting mutable directories alone is
not equivalent to immutability.

The attestation must bind the exact source commit and source tree, lockfile,
dependency material, build command and environment, approved builder and image
identities, runner digest, scenario snapshot and identifier, execution policy,
exit status, and digests of the reports and captures. It must be authenticated
by the approved builder after both build and run. The verifier must check the
attestation independently, recompute available artifact digests, and retain the
existing exact scenario, steps, outline, cleanup and revision checks. The
existing sealed runner can help bind the built executable to execution; it is
not by itself an authenticated attestation.

The policy must specify how dependency material is authenticated. The current
registry archive check and partial Git vendor checks are useful but insufficient
to claim complete Git manifest provenance. A build image that consumes source
archives instead of normalized Git vendor manifests is one possible design; its
actual Cargo resolution and workspace behavior must be tested before adoption.
Do not infer a complete dependency proof from a successful partial check.

## Integration and evidence gates

1. Define and review the builder trust policy and an immutable image construction
   and update process. Select a supported display/renderer configuration. Keep
   existing receipts historical while that process is unavailable.
2. Build one real pinned application runner using only approved, immutable
   inputs. Verify that caller Rust/Cargo overrides, ignored files, mutable host
   tools, network access and dependency substitutions cannot affect its build.
   Exercise transient input substitution during the build, not just changes
   visible before or after it.
3. Execute that exact runner against one pinned scenario under the approved
   runtime. Test runner replacement, scenario substitution, display/runtime
   substitution, fabricated reports and changed captures. Check native window
   interaction and cleanup. Verify all example rows for an outline.
4. Produce an authenticated chain and independently verify it against the
   external trust policy. Reject unknown schema versions, unapproved builder or
   image identities, altered artifacts, mismatched runner and scenario hashes,
   failed runs and expired or revoked authority. A schema field or `reusable`
   flag alone must never authorize verification.
5. Connect that checked chain to the production capture and receipt verifier.
   Keep historical receipts stale. Test the full ledger transition with a valid
   attested receipt and negative cases for every boundary above. Keep selective
   cross-commit reuse disabled until #3974 has its own complete dependency and
   runtime policy.

No current check establishes all these conditions. In particular, a pinned
runner hash binds bytes between the build observation and run; it does not show
that the bytes were built from approved inputs. No native UI acceptance result
or reusable receipt is claimed by this document.
