# Testing 01 — Core & integration testing

*How the non-renderer 90% is tested — the wide base of the pyramid. This is the easy
part precisely because of ADR-0002/0008 (cgo-free, GPU-free core): `go test` covers
it, fast and deterministic. The emphasis here is on the test *forms* that suit an
LLM author and a parametric-CAD domain.*

## Pure-Go unit tests (the base)

`math`, `kernel`, `model`, `param`, `sketch`, `solve` are ordinary Go packages with
no cgo and no GPU. `CGO_ENABLED=0 go test ./...` runs the whole domain in seconds on
any runner. Standard table-driven unit tests carry the bulk; the domain-specific
forms below add the leverage.

## Property-based & metamorphic tests (kernel & solvers)

The kernel and solvers have **mathematical invariants** an LLM can assert without
hand-computing expected outputs — the most cost-effective tests in the project:

| Subsystem | Property / metamorphic relation |
|---|---|
| **kernel booleans** | `A ∪ A = A`; `A − A = ∅`; `(A ∪ B) volume ≤ vol A + vol B`; commutativity of `∪`/`∩`; double-negation `A − (A − B)` |
| **kernel geometry** | tessellation volume → true volume as tolerance→0; closed body is watertight; face areas sum sanity |
| **mass properties** | inertia of a known primitive = analytic; COM of a symmetric body on its axis |
| **constraint solver** | a fully-constrained sketch ⇒ 0 DOF; re-solve is **idempotent**; warm-start from solution ⇒ no movement; adding a redundant constraint ⇒ flagged, geometry unchanged |
| **assembly solver** | grounded part is fixed; a mate makes the residual ≈ 0; over-constraint detected |
| **feature recompute** | recompute is **idempotent** (recompute twice ⇒ identical bodies + lineage); order-independent for independent branches |
| **parameters/units** | `parse(format(q)) == q`; dimensional errors rejected; cycle ⇒ sick not hang |

Property tests (with shrinking, e.g. `gopter`/`rapid`) generate random valid inputs
and assert the invariant — catching edge cases neither the author nor the LLM would
enumerate (degenerate triangles, tiny/huge scales, coincident geometry). They are
ideal for an LLM: it writes the *invariant*, the framework finds the counterexample.

## Reference-key tests (the make-or-break seam)

The hardest correctness property in the system (parametric-cad §7, ADR-0010) gets
dedicated tests, all pure-Go and deterministic:

- **survives recompute:** mint a key to a face, change an upstream parameter, recompute,
  assert the key **re-resolves to the topologically-corresponding face**.
- **fails honestly:** delete the feature that created a referenced face, recompute,
  assert resolve returns `not found` and the consumer goes **sick** (not crash).
- **survives reload:** key → save → load → resolve. Round-trips through the context
  serialization (core/05).

These are exactly the tests that catch the "edits silently lose references" failure
mode — and they need no GPU and no oracle, just assertions on which entity resolves.

## Round-trip & golden-model tests (persistence)

- **Serialization round-trip:** build a model → save (zip package) → load → assert the
  loaded model recomputes to **identical bodies** and equal parameters/features. The
  truth is the recipe, so compare recipes + recomputed geometry, not raw bytes.
- **Golden model files:** a corpus of `.obk` packages (committed) that must continue to
  open, migrate, and recompute across versions — guards the migration pipeline (core/05).
- **Backward-compat:** old-version goldens open via migration and produce expected output.

## Command / undo tests

Because every edit is a command (core/06), undo correctness is directly assertable:

- **`Apply` then `Revert` ⇒ document identical** to before (deep equality of the recipe).
- **redo = re-apply**; **undo/redo stack** invariants; **composite/batch** undoes as one.
- Property test: a random sequence of edits, fully undone, returns to the initial state.

## Event / veto tests

Subscribe in-test to the typed bus (core/06): assert the right before/after events
fire with the right payloads, and that a `Veto(reason)` from a before-handler actually
cancels the operation (e.g. a vetoed close leaves the document open).

## Integration tests (headless, CGO_ENABLED=0)

Wire real subsystems together without a GPU, via a **headless `Runtime`** (core/00,
null renderer):

- **model ↔ persistence ↔ recompute:** create a part through commands, save, reload,
  recompute, assert geometry — the full document lifecycle.
- **the `/api` dogfood suite (core/07, ADR-0018):** drive a complete part/assembly
  build **entirely through the public surface** — the `api/client` typed client over
  the host's `api/wire` methods — and assert the result equals the in-proc build.
  This simultaneously tests the API *and* proves it is complete — the dogfood
  principle (realtime-3d §12) as an executable check. The same suite is the
  conformance test third-party add-ins can run against. (A gRPC transport behind the
  same `api/wire` surface is a deferred future, ADR-0003/0016.)
- **automation (M15):** run an iLogic rule / iPart member generation and assert the
  reconfigured model — automation is a client of the API, so this is an integration
  test of the API surface (apps/01).

## Determinism for the core

The pure-Go core is deterministic by construction (no GPU, no wall-clock in recompute,
seeded RNG where used), so failures **reproduce every run** — the property an LLM most
needs to fix a bug. Recompute runs on the worker pool, but results are order-
independent for independent branches (a tested property above) and the document is
mutated only on the main goroutine (ADR-0007), so concurrency introduces no test
flakiness. Run the suite with `-race` in CI to pin the few genuinely concurrent seams.

## Where the renderer hands back to the core

The most important integration boundary for *renderer* debugging: when an image is
wrong, the structured diff (testing/00) often points **below the GPU line** — e.g.
"ID buffer empty ⇒ draw list empty." At that point the failure is a `null`-backend
draw-list assertion (core/08, ADR-0014) — a pure-Go unit test. The renderer's hardest
bugs are deliberately routed back into the part of the codebase that is easiest to
test and easiest for an LLM to fix.
