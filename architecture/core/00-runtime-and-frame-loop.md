# Core 00 — Runtime mediator & frame loop

*Modernizes: the COM `Application` singleton (M00-F04) and the synchronous COM
event/transaction cadence (M04). Applies realtime-3d skill §1–2, §13.*

## The runtime mediator (kills the `Application` singleton)

The COM model had a global `Application` object reachable from every object's
`.Application`/`.Parent`. We replace it with **one runtime mediator owned by the
app and passed explicitly** — no globals, no service locator.

```go
// runtime/runtime.go
type Runtime struct {
    Clock      *Clock          // runtime seconds + frame counter
    Jobs       *WorkerPool     // bounded goroutine pool (recompute, tessellation, cull)
    Frame      *Scheduler      // ordered per-phase callback registry
    Window     platform.Window // ADR-0008 edge
    Input      *input.State
    Renderer   *renderer.Renderer
    Scene      *scene.Graph    // viewport entities + transform DAG
    UI         *ui.Shell       // ImGui shell (ADR-0004)
    Docs       *model.Workspace// open documents (replaces Documents collection)
    Selection  *Selection
    Commands   *command.History// undo/redo (ADR-0006/core-06)
    Events     *event.Bus      // typed observers (core-06)
    Registry   *registry.Set   // self-registered features/workspaces/commands
    AddIns     *addins.Host    // out-of-proc gRPC supervisor (ADR-0003)
    Log        *slog.Logger
}
```

Rules (realtime-3d §1):
- The mediator **owns and exposes** subsystems; it holds **no domain logic**. It
  is a directory, not a brain.
- Multiple `Runtime`s can coexist (a headless one for tests/CI/thumbnails, a
  second window). Nothing is global, so this is free — the same property the COM
  `ApprenticeServer` headless mode hacked around.
- Subsystems that need a peer take it as a **narrow typed parameter**, not the
  whole runtime, where the dependency is small. The runtime is for sharing; pass
  slivers where you can.

> **Model tree vs. mediator.** The *document* object graph (document → component
> definition → feature) keeps explicit `parent` links because it genuinely is a
> tree and the domain navigates it. What we removed is the *global* `Application`
> back-pointer on every object. Domain objects reach services via the document's
> `*Workspace`/`*Runtime` handle threaded at the document boundary, not a singleton.

## The frame loop & ordered phases

One loop, explicit named phases in a fixed order (realtime-3d §2), adapted for CAD:

```
each frame (dt seconds):
  1. deferred      # run queued "next frame / after N / after T" callbacks
  2. input         # drain window/input events → input.State, route to interaction
  3. interaction   # active command/manipulator consumes input (drag, pick, snap)
  4. commands      # apply committed edits → mutate document, mark dirty (DAG)
  5. recompute-sync# swap in any COMPLETED async recompute/tessellation results
  6. transforms    # parallel recompute of dirty world matrices (scene DAG)
  7. cull+build    # per-camera cull the render queue, build draw lists
  8. ui            # ImGui: build the shell from current model state
  9. render        # record + submit Vulkan; present
```

Key points:
- **Recompute is NOT a phase** — it runs on the job pool (ADR-0007). Phase 5 only
  *adopts* finished results at a safe boundary. The loop never stalls on a rebuild.
- **Destruction is deferred** (realtime-3d §2): deleting an occurrence/feature
  queues the removal; it executes at phase 1, never mid-iteration.
- **`dt` in seconds** flows to interaction/animation (constraint-drive, exploded-
  view playback). A monotonic clock + frame counter live on the runtime.
- **Subscribe/unsubscribe by handle**: `id := rt.Frame.On(PhaseUpdate, fn);
  rt.Frame.Remove(id)` — O(1), unambiguous.

## The worker pool (the concurrency spine)

A single bounded `WorkerPool` (realtime-3d §13) routes all embarrassingly-parallel
work so we never spawn ad-hoc goroutines in hot paths:

- parallel **feature recompute** of independent DAG branches (ADR-0007),
- parallel **tessellation** of changed bodies,
- parallel **transform** world-matrix recompute (scene DAG, phase 6),
- parallel **culling**, parallel **import/export**.

Workers operate on **immutable inputs** and return results; the document is mutated
only on the main goroutine at phase boundaries (ADR-0007 invariant) — so the model
needs **no locks**, the hardest concurrency bugs are designed out.

## Performance discipline (realtime-3d §13)

- **Near-zero steady-state allocation** in phases 6–9; pre-size and reuse draw-list
  and command buffers; pool per-frame transient structs.
- **Tracing regions**: `defer rt.Trace("scene.recomputeTransforms").End()` compiles
  to a no-op in release; gives a free frame timeline in debug.
- GC tuned (`GOGC`, optionally `SetMemoryLimit`) so collections land between frames,
  not within them; the async-recompute split keeps the big allocator (the kernel)
  off the frame thread entirely.

## What this replaces, concretely

| COM (old) | Here (new) |
|---|---|
| `Application` global, `obj.Application` everywhere | `*Runtime` mediator, passed |
| synchronous edit→recompute→UI freeze | phases 4→(async)→5; loop stays live |
| `ApprenticeServer` separate headless API | a headless `Runtime` (null renderer) |
| ad-hoc COM apartment threading | one main goroutine + a bounded worker pool |
