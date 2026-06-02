# Testing 02 — Performance, UI & add-in conformance

*The test surfaces the first two testing docs did not cover: performance/regression
(the realtime-3d skill makes frame budget and allocation first-class — §2, §13), the
ImGui UI shell (core/09), and the out-of-process add-in contract (ADR-0003). All
remain LLM-friendly: numeric thresholds and assertions, not visual judgment.*

## Performance & regression testing

A real-time loop means performance is **correctness**: a frame-budget regression is a
bug. These are gated numerically, so an LLM sees "draw-list build 0.9ms → 3.1ms,
budget 1.0ms" rather than "feels laggy."

- **Microbenchmarks (`testing.B`)** on the hot paths the frame loop and recompute
  depend on: transform world-matrix recompute (scene DAG), culling, draw-list build,
  tessellation, boolean ops, the constraint solvers, expression eval. Tracked over
  time with **benchstat**; CI fails a hot path that regresses past a threshold.
- **Allocation budgets** (realtime-3d §13: near-zero steady-state allocation). Use
  `testing.AllocsPerRun` / `-benchmem` to **assert `allocs == 0`** for the per-frame
  hot loops (render-queue build, transform recompute, culling). A new heap allocation
  in a frame phase fails CI — the GC-pause defense is a test, not a hope.
- **Frame-budget assertions** using the built-in tracing regions (core/00): in a
  headless deterministic run, assert each phase's time stays within its slice of the
  16.6ms budget for a reference scene.
- **Recompute-scaling (metamorphic perf):** editing the *last* feature of a
  500-feature part must recompute O(dirty-tail), not O(n) — assert the rebuild touches
  a bounded set, pinning the rollback-replay invariant (ADR-0010) as a *performance*
  property, not just a correctness one.
- **Large-assembly soak:** load a 10k-occurrence assembly (one mesh, N transforms —
  the GPU-instancing path, assembly/00), run N frames, assert **no memory growth**
  (leak detection) and stable frame time. Catches per-frame leaks the unit tests miss.
- **`-race` in CI** on the genuinely concurrent seams: the worker pool, and the async
  recompute → frame-boundary result swap (ADR-0007). The "document mutated only on the
  main goroutine" invariant means few such seams — pin them.

## UI (ImGui) testing — test the logic, not the pixels

The shell is immediate-mode, **built from model state each frame** (core/09), and
state lives in the **model, not the UI**. So most "UI bugs" are model/command bugs
already caught by core tests (testing/01). The genuinely UI-specific surface is small
and **pure-Go testable**:

- **Reflection-driven inspector generation** is the highest-leverage UI code (one path
  generates every feature's edit panel, core/09). Test it directly: given a
  `Definition` struct + field tags, assert the **generated widget descriptors**
  (labels, units, clamp ranges, enum choices, pick filters, visibility). No GPU — it's
  reflection → descriptors. A wrong tag mapping is a unit-test failure.
- **Ribbon/browser generation:** assert the `registry.Commands` → ribbon mapping
  (which buttons, in which category, with which enable-predicate) and the document →
  browser-tree mapping (feature/occurrence nodes) on the headless path.
- **Command dispatch:** simulate an action (button/menu/hotkey) → assert it runs the
  right command and pushes the expected entry to history (core/06), so the edit is
  undoable. Pure logic, no rendering.
- **Interaction flows (integration):** the **Dear ImGui Test Engine** scripts headless
  flows — "start extrude, pick profile, set distance=10, OK" — against the offscreen
  backend, asserting the resulting **model state** (an extrude feature with distance
  10), not pixels. This is the UI integration layer.
- **Chrome visual regression (optional, low priority):** the ImGui pass can be captured
  by the offscreen backend and diffed against a golden via the image oracle
  (testing/00) — but UI pixels are deliberately *not* pinned tightly; the logic tests
  above carry the weight, and ImGui's own rendering is upstream-tested.

The takeaway for an LLM author: **UI correctness reduces almost entirely to pure-Go
assertions on descriptor generation and command dispatch** — the one subsystem you
might expect to need a human's eyes barely does, because the architecture keeps state
in the model and generates the UI from it.

## Add-in / API conformance testing

The add-in contract (ADR-0018) needs its own tests beyond the dogfood suite
(testing/01), covering the *boundary* properties COM never had:

- **Contract / schema compatibility (CI gate):** a check on the `api/wire` DTOs and
  method-name constants that JSON fields and methods are never removed or renamed and
  changes stay additive, so an old add-in keeps working against a new host. (The same
  rule maps to protobuf field numbers if/when the gRPC transport lands.)
- **Conformance kit:** a versioned set of `api/client` interactions with expected
  results that **third-party add-ins run to certify** their integration — the same
  suite the host uses to dogfood the API doubles as the public conformance test.
- **Crash isolation (fault injection), out-of-proc only:** the in-process C-ABI
  transport (ADR-0016) shares the host address space and has **no** crash isolation;
  a crashing add-in can take the host down. When the deferred out-of-process gRPC
  transport (ADR-0003) lands, this test kills an add-in process mid-call and asserts
  the **host survives**, detects the dropped connection, and degrades gracefully.
- **Veto deadline:** an add-in that never answers a vetoable before-event → assert the
  host **proceeds after the deadline** (ADR-0003/core-06), so no add-in can hang the
  host — the COM footgun, tested.
- **Permission enforcement:** an add-in calling a service outside its manifest grant →
  assert **denied** (ADR-0003 capability model).
- **Backward/forward compat:** an add-in built against an older proto, run against the
  current host, still functions.

## Where these sit in the pyramid

- Microbenchmarks / allocation budgets / UI-logic tests → **base** (fast, GPU-free,
  every change).
- Frame-budget / recompute-scaling / interaction-flow / conformance → **middle**
  (headless, PR).
- Large-assembly soak / chrome visual regression → **top** (nightly, with the
  Blender-oracle goldens).

All three surfaces keep the project's testing throughline: **failures are numbers and
locations**, so the LLM that wrote the code can fix it.
