# Verification, compatibility, and release

This document defines future implementation gates. The planning change has
no Rust source or Cargo workspace to build. Its own checks are Markdown lint,
document links, source-reference checks, and `git diff --check`.

## Behavior ledger

R0 creates a checked-in, machine-readable ledger. Each behavior row records:

- Stable ID, source revision, source location, and related GitHub issues.
- Public method, command, document field, feature, or user workflow it covers.
- Existing Go tests and fixtures, plus whether their premises are valid.
- Target Rust module, milestone, API revision, kernel capability IDs, and
  required operating-system and device profiles.
- Rust test IDs, comparison/oracle inputs, expected results, and evidence files.
- Status: inventoried, blocked, implemented, verified, or explicitly excluded.
  Exclusions need a reason and the scope decision that permits them.

Fail CI when a new baseline item is unclassified or a claimed verified item
lacks evidence for the current relevant revision. Report behavioral coverage
separately from source coverage. A passing test suite with half the commands
missing is not a completed port.

Keep Go and Rust API generation in the API repository. In this application,
check that each required API method has the expected host implementation,
mutation policy, contract tests, and script exposure. Do not create a competing
schema or declare missing methods implemented by returning a generic error.

## Comparison rules

Use the Go application for observed application behavior and the canonical
API contract for public serialization and method shapes. Use pinned OCCT or
SolveSpace evidence for geometry and solver results. A known Go defect is a
regression input, not the expected geometric answer.

Compare exact values where the contract requires them: IDs, key bytes,
enumerations, field presence, command effects, event ordering, and recipe
meaning. Compare geometry against justified, recorded metric tolerances and
per-face topology/geometry assertions. Never increase a tolerance to obtain
a passing comparison.

Go and Martensite can represent a valid result with different internal face
order or decomposition. Define each geometric correspondence from provenance
and analytic roles before comparing it. Do not sort away a missing face,
silently ignore a topology difference, or require arbitrary pointer order.
Required public identity must still survive the change of representation.

For the same Rust build and inputs, require deterministic ordered output and
cross-platform stability. Explicitly test tie-breaks, FMA-sensitive arithmetic,
signed zero, matrix conventions, integer widths, paths, locale, and endian
encoding. Use byte comparisons for canonical artifacts that promise byte
stability. GPU captures need controlled profiles and explicit image checks;
their pixels are not topology evidence.

## Required test layers

| Layer | Required checks |
| --- | --- |
| Module | Every new function's behavior, errors, and invariants; dependency injection; named external-I/O fakes. Avoid tests that only repeat the implementation. |
| Contract | Applicable Rust trait implementations, Go/Rust typed client calls into the real host, absent/null/empty values, enums, raw JSON, errors, and all registered method effects. |
| Model | Parameter dependencies, solver rank/DOF, cycles, feature health, suppression, clean-prefix reuse, partial failures, and property/attribute retention. |
| Kernel | Validity levels, analytic per-face oracle checks, operation history, generated degenerate inputs, exact predicates, tolerances, and deterministic results. |
| Persistence | Go-produced fixtures read by Rust; Rust writes read by Go when using the unchanged schema; semantic round trips for all document kinds, projects, resources, and keys. |
| Concurrency | Queue saturation, request timeout, cancellation, out-of-order worker completion, stale generations, close/undo during recompute, and shutdown with active add-ins. |
| FFI | C layout/signatures, allocation ownership, error buffers, invalid lengths, missing/versioned exports, callback lifetime, reentrancy, notification and automation calls. |
| Live GUI | Foreground app driven through MCPBridge; rendered output, picked reference, model state, dialogs, editing, and repeated resource lifecycle. |
| Distribution | Clean installed packages, runtime assets, shared-library lookup, file associations, update/rollback, licenses, and dependency version reporting. |

Geometry changes require all three kernel test layers: invariants, hard oracle
assertions, and generated degenerate/near-degenerate cases. The adapter must
propagate defects to feature health, automation, and UI. Validate whole-body
properties as smoke tests in addition to the per-face proof.

When kernel work is needed, its change report includes the net recognizers,
tolerance constants, fallback sites, and geometry type assertions required by
the ground rules. An application adapter cannot conceal increases or move a
type-specific workaround out of the kernel audit.

## First integrated live test

Start the foreground Rust GUI with the Go MCPBridge and both language test
add-ins. Use the public typed API and record the following sequence:

1. Create a part with named parameters and formulas for derived dimensions.
2. Create one sketch for the extrusion. Apply dimensional and geometric
   constraints until DOF is zero. Verify both constraint status and geometry.
3. Extrude with a parameter as its dimension. Verify validity, analytic face
   types, loop counts, and expected properties through model checks.
4. Select a generated face and record its persistent key. Create a dependent
   sketch/feature when its R3 support is present.
5. Change an upstream parameter while recompute is in flight. Confirm that
   the latest generation wins and that selection resolves to the right face.
6. Undo, redo, suppress, restore, save to a new `.opd`, close, and reopen.
   Verify the recipe, constraints, dependent references, and feature health.
7. Force a failed edit and a lost reference. Confirm a clear diagnostic,
   transaction behavior, and isolation of affected dependent features.
8. Capture and inspect the viewport with the MCP screenshot API. Check the
   displayed shape, edges, selection, overlays, and UI state separately.
9. Repeat activation/deactivation at least 20 times per test add-in, then
   shut down with no pending callbacks or retained host buffers.

Store the source/dependency SHAs, platform, GPU/driver, inputs, assertions,
logs, and captures with the test receipt. Screenshots supplement numeric and
identity assertions. A screenshot alone cannot prove a valid solid.

Before each GUI implementation PR, add a foreground MCPBridge test for the
changed path and inspect its screenshot. If R1V reaches a GUI PR before R4,
bring forward the minimum C ABI/MCP host needed for that test, or keep the
window proof local until that host is ready. Do not waive the live-test gate.

## Full local gate and CI

R0 must add **one command: `make gate`**. Its implementation derives the crate
set from Cargo metadata and the fixture-module set from the tree. It includes
the CLI, GUI, native bindings, examples, integration tests, and add-in test
libraries; no default workspace subset may omit them.

The complete gate includes:

- Locked builds, `cargo fmt --all -- --check`, Clippy with warnings denied,
  workspace tests, documentation tests, and generated-file drift checks.
- Architecture imports, API isolation, required host-method coverage, SPDX,
  Markdown lint, docs/examples, dependency notices, and declared license checks.
- Go and Rust add-in fixtures against the Rust host. Go is a test tool here,
  not a runtime dependency of the Rust application.
- All required local UI/GPU and live checks on a real display or supported
  virtual display. Missing display, ICD, library, or dataset is a named failure.
- Source coverage **greater than 80%** and duplication **less than 3%**.
  Report per-crate values as well as totals so one large crate cannot hide
  an untested host module. Explain generated/vendor exclusions explicitly.
- A cross-platform stability review and a comment/coherence review before
  the first push. Keep required follow-up work in one batch when possible.

Fast commands may run selected tests during development. They must not be
named or reported as the full gate. Gate obligations grow with the branch:
before a GUI exists there is no GUI pass to claim; as soon as the GUI is added
the discovered checks must build and exercise it. Required target-OS checks
also run in native CI jobs; a Linux local pass cannot stand in for them.

Run the API repository's two-language gate when changing its contract. Run
Martensite's required inventory, oracle, and runtime checks when changing the
kernel. Application CI records these dependency receipts and adds its own
integration tests. A version bump alone does not accept a dependency update.

## Platform and performance matrix

| Profile | Evidence |
| --- | --- |
| Linux x86-64 with software Vulkan and virtual/real display | Complete deterministic CI suite, validation layers, C ABI add-ins, UI lifecycle, and capture. |
| Linux with a native GPU | Foreground model/selection/render tests, resize/lifecycle stress, and performance measurements. |
| Windows x86-64 | Native builds and runtime tests, DLL lookup/lifetime, Unicode paths, graphics, package installation and update. |
| macOS arm64 | Native builds and runtime tests, Vulkan portability/native window composition, dylib handling, package and input behavior. |
| Other architectures shipped by the Go release | R0 inventories them; add native correctness/runtime coverage before shipping equivalent Rust packages. |

Record median and tail latency, memory peak, startup, save/load, solve,
recompute, tessellation, selection, frame time, and add-in round trips. Compare
the same inputs, hardware, build profile, and measurement method. Freeze
per-workload limits after the Go baseline and the first Rust workflow are
measured; do not invent a port-wide speedup percentage.

Include small parts, large feature histories, repeated assembly instances,
imported curved faces, and heavy overlays. Separate CPU, GPU, cold-start, and
warm-cache results. A faster wrong result fails correctness. Keep performance
evidence with the same revision as the functional evidence.

## Release and retirement

Early `rust` builds use a distinct preview channel and install location. Do
not change the current Go release channel as part of workspace setup. Bundle
the API version, Martensite revision, toolchain, document schema, reference-key
scheme, and C ABI compatibility information in each application build.

Pin dependencies to fetchable commits or release tags and commit `Cargo.lock`.
Keep optional local development overrides untracked. Verify packages in an
environment without source siblings, caches that hide missing files, or an
installed Go application. Inventory retained native UI/runtime dependencies
and license notices; preserve the Apache API versus GPL application split.

The cutover issue owns the deletion of the Go production build/release path
and any temporary application adapter. Two kernels in one shipped application
are not a migration strategy. The Go reference executable may remain outside
the product in comparison tests; its revision and purpose stay explicit.

Before the stable switch, all required baseline rows must be verified, both
add-in languages must pass, and document/key migration must preserve user
work. Test a failed upgrade with the original file intact and a usable prior
installation. Document schema changes must not destroy the rollback copy.

The final review must state which existing Go roadmap items remain unfinished
and which deliberate behavior changes have their own decisions. A port can
preserve an unfinished baseline feature without falsely marking that feature
complete. It cannot omit working baseline behavior and still claim full parity.
