# Testing strategy

*A senior-test-engineer reevaluation of the architecture, written for a codebase
**authored and debugged primarily by an LLM (Claude)**. Implements ADR-0014. The
renderer gets its own deep treatment — [testing/00](00-renderer-oracle-pipeline.md) —
because it is the hardest surface to test and the hardest for an LLM to debug. Core
and integration testing: [testing/01](01-core-and-integration-testing.md).
Performance, UI, and add-in conformance: [testing/02](02-performance-ui-and-conformance.md).*

## The guiding principle: give the model a numeric, localized signal

An LLM debugs well when failure is a **wrong value with a location** and badly when
failure is "it looks wrong." Every testing decision below exists to convert the
latter into the former. Two corollaries:

- **Prefer oracles that produce explainable numbers** (analytic expectations, CPU
  references, invariances) over human visual judgment.
- **Localize failures** — to a subsystem, a pass, a region, a parameter — so the model
  edits the right code instead of guessing.

## The architecture is already highly testable — by earlier decisions

Most of this codebase was made testable by decisions taken for other reasons. The
testability ledger:

| Subsystem | Why it's already testable | Decision |
|---|---|---|
| `math`, `kernel`, `model`, `param`, `sketch`, `solve` | **cgo-free, no GPU, no OS** — `go test` anywhere, fast | ADR-0002, ADR-0008, core/01 |
| feature recompute | **pure function** `(snapshot) → (bodies, health)`, no global state | ADR-0010, modeling/01 |
| constraint solvers | pure `System.Solve()`, deterministic, headless | ADR-0009, ADR-0011 |
| edits / undo | **command objects** with `Apply`/`Revert` — assert round-trips | core/06 |
| events | typed bus — subscribe in a test, assert emissions/veto | core/06 |
| identity | reference keys are **serializable values** — assert resolve across recompute | core/05 |
| persistence | zip package + codecs — **round-trip** save→load equality | core/05 |
| renderer (pre-GPU) | **draw-call-as-data** — build draw lists on CPU, assert | core/08, ADR-0014 |
| public API | the `/api` contract (`api/client` over `api/wire`) is exercised in tests = **dogfood** | core/07, ADR-0018 |
| platform/GPU edge | isolated behind interfaces; `null`/`offscreen` backends | ADR-0008, ADR-0014 |

The deliberate consequence (ADR-0002/0008): **the valuable 90% — the entire domain
and kernel — runs `CGO_ENABLED=0` in CI with no GPU**, so the bulk of the test suite
is fast, deterministic, and hardware-independent. The renderer is the exception this
strategy is built to handle.

## The test pyramid for this app

```
        ╱ Blender-oracle full-PBR goldens ╲          ← few; CI; perceptual tolerance
       ╱  end-to-end (model → render) goldens ╲       ← few; offscreen on llvmpipe
      ╱   renderer differential / AOV tests     ╲     ← analytic + CPU-ref + metamorphic
     ╱    integration (model↔persistence↔api)    ╲    ← headless, CGO_ENABLED=0
    ╱  property & metamorphic (kernel, solver)     ╲  ← invariants, no oracle needed
   ╱  unit tests (math, kernel, model, param, …)    ╲ ← the wide base; fast; GPU-free
  ╶──────────────────────────────────────────────────╴
```

- **Wide base — pure-Go unit + property tests** (testing/01): the kernel, solvers,
  feature engine, parameters. Most bugs are caught here, cheaply, with exact failures.
- **Middle — renderer differential tests** (testing/00): analytic, CPU-reference, and
  metamorphic oracles on the `offscreen`/`null` backends. No human in the loop.
- **Narrow top — Blender-oracle + end-to-end goldens** (testing/00): full-PBR ground
  truth and model→image pipelines, in CI, with perceptual tolerance and a human
  **bless** step for intentional changes.

## CI shape

- **Tier 1 (every change, minutes):** `CGO_ENABLED=0 go test ./math/... ./kernel/...
  ./model/... ./solve/...` + renderer `null`-backend unit tests + analytic/metamorphic
  renderer tests on **software Vulkan (Mesa lavapipe)**. No GPU, deterministic.
- **Tier 2 (PR):** CPU-reference renderer differentials; integration goldens; the
  gRPC-API dogfood suite; validation-layers-on renderer runs.
- **Tier 3 (nightly / on renderer changes):** Blender-oracle full-PBR goldens
  (containerized, pinned Blender); large-scene perf/soak; the **bless** workflow to
  review and update goldens when a visual change is intentional.

## What "Claude debugs this" demands, concretely

- **Validation layers + SPIR-V validation are hard gates** — API/shader misuse fails
  loudly at the source, not as silent visual corruption.
- **Failures emit structured numeric reports** (per-pass score, failing tier, error
  region) — never "compare these two PNGs by eye."
- **Small, single-purpose shaders & passes** — a failing golden points at one shader.
- **Metamorphic relations everywhere** — invariances Claude can assert without any
  reference image (translate/rotate/scale/resolution/instancing equivalence; recompute
  idempotence; save/load identity). These are the cheapest, most LLM-authorable tests.
- **A deterministic inner loop** — software Vulkan + determinism mode, so a test that
  passes locally passes in CI, and a regression reproduces every run.
