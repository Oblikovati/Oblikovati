# Kernel and host rules

## Application extension and refactoring

Oblikovati began as an Oblikovati GoLang port, instead of using its own kernel, it is preserving the
application layer features and building on top of a new kernel, Martensite, which by itself started
as a port of OCCT, Martensite API now evolves to fit the needs of Oblikovati.

### Change and replacement

- Follow semantic versioning. A breaking change needs a version bump and changelog migration notes.
- Separate structural refactors from numerical or behavioral changes.
- Pin current behavior with owned fixtures before a refactor; a refactor passes them unchanged.
- A behavior change needs independent expected results, a kernel decision, and a new behavior version.
- Keep earlier behavior selectable for replay until a recorded migration.
- Reuse existing adaptors, builders, history modules, and mesh factories.
- Give each replacement a deletion task, corpus gate, and one-milestone migration limit.
- Prove one complete operation before extending a replacement engine.
- Do not add Go-kernel code or production fallbacks; check Go-derived work against its known defects.

### Operation result contracts

- Return completion, diagnostics, validation coverage, history, and achieved error together.
- Return typed errors with operation, entities, failure class, and cause; never only a flag or a string.
- Distinguish completed empty results from failed, cancelled, or incomplete computations.
- Validate topology, geometry consistency, self-intersection, and tolerance consistency in order.
- Select validation invariants for the declared body kind.
- Make the check mode explicit; a sampled pass is not an exact pass.
- Offer local checks of changed entities; keep the full ordered check for publication.
- Treat missing or inconclusive required checks as incomplete, never as valid.
- Keep invalid and partial geometry out of successful operation results.
- Express history as deleted, merged, created, split, or derived relations with all parents.
- Compose operation, copy, and healing history; leave document naming policy to the application.
- Perform healing explicitly on a copy and return its identity remap.

### Geometry and numerical policy

- Use one general pipeline per operation; keep conditioned fast paths inside it.
- Do not add shape-specific retry ladders or silent faceted replacements.
- Never use tessellation for modelling decisions or acceptance evidence.
- Take area and volume evidence from error-bounded integration, with its error.
- Keep spatial, parameter, angular, modelling, and approximation tolerances distinct.
- Report requested limits, effective settings (including clamps), and measured error separately.
- Tag each measurement as a certified bound, a sampled estimate, or unknown.
- Give tolerance caps explicit no-limit, report, and refuse modes; never default to no limit.
- Do not widen limits or reduce recorded error to satisfy an acceptance check.
- Use exact or filtered signs for topology decisions; keep metric tolerance tests separate.
- Compute shared incidences once and reuse their results.
- Never derive identity from returned order, coordinate rank, or proximity.
- Certify pruning bounds and root choices; report exhausted search as incomplete.

### Ownership and thread safety

- Published results are immutable, `Send`, and `Sync`; reading them takes no lock.
- Mutate only in operation-local builders, before publication.
- Make operations non-destructive by default; test input isolation after success, failure,
  cancellation, and source mutation.
- Define ownership for all geometry, topology, caches, and meshes reachable from a result.
- Do not treat `Clone`, `Arc`, locks, or `&Shape` as proof of isolation.
- Keep no thread-local or global mutable algorithm state; pass state in an operation context.
- A result must not depend on the thread, earlier work on it, or concurrent operations.
- Make operation objects `Send`; avoid `Rc` and `RefCell` in operation state.
- Give each `unsafe impl Send/Sync` a safety comment and a Miri or loom test.
- Assert `Send`/`Sync` at compile time for each public type, or document why it is thread-confined.
- Key reusable results by input revisions, options, and tolerance policy.
- Merge parallel results in a total order; no output may depend on hash iteration order.
- Measure copy, allocation, and lock costs before redesigning storage.

### Verification and completion

- Use independent oracles: closed forms, error-bounded integration, Monte Carlo membership,
  invariants, and reviewed baselines.
- Change a baseline only by a reviewed update that states the reason and behavior version.
- Require invariants, independent expected results, and generated degenerate cases for changed algorithms.
- Check per-face geometry, trims, tolerances, and history; volume alone is insufficient.
- Compare two processes byte for byte; compare multi-thread with single-thread results.
- Run concurrency stress tests and ThreadSanitizer for code that claims thread safety.
- Record revisions, options, platforms, oracle kinds, executed tests, and exclusions.
- Keep kernel acceptance and application acceptance separate.
- Benchmark identical accepted workloads against the previous release; report time and peak memory.
- Remove superseded production paths when their replacement gate passes.
- Do not weaken tests, widen tolerances, or hide failures to complete a migration.
