# Express assembly loading design

Issue #3949 tracks the native acceptance journey. Express loading is not yet
implemented. The Open dialog declines it; it must not load a full assembly and
label the result Express.

## State and contract

The public API needs a typed Express open request, a load-state query, and a
Load Full operation. Define these in Oblikovati.API before the host routes them.
The local Inventor reference exposes `AssemblyDocument.IsOpenExpress`: a
loaded Express assembly may transition to Full; returning to Express requires
close and reopen. A different open mode for an already open path must be
resolved explicitly rather than silently reusing an incompatible owner.

Express holds the authored assembly identity and a validated display cache. It
must not expose cached triangles as analytic bodies, face references, feature
health, measurements, or inputs to modelling operations. Operations that need
full component content must return a named decline. Load Full must prepare and
validate the reference graph and analytic replay before publishing the change;
a failure must retain the Express document, its visible cache and active tab.

## Display cache and provenance

At full save, generate the cache from the same validated scene and configuration
as the saved assembly. Store a versioned presentation-only stream in the
assembly package, atomically with its model recipe. Include the assembly recipe,
occurrence transforms and visibility, material and appearance, active model
state, meshing policy and renderer-independent triangle data. Reject non-finite
coordinates or colours, malformed records and incomplete presentation results.

Record a deterministic, complete transitive dependency manifest. Each entry
must identify its owner and reference, resolved source identity and content
digest. Verify that the manifest matches the authored nested graph, including
suppression and configuration; a list of only top-level filenames is not enough.
Unsaved modified dependencies, unresolved references, cycles, stale data, absent
caches and unsupported versions require explicit declines. Reads used to verify
hashes must be the same bytes retained for the subsequent operation, so a file
replacement cannot silently validate one version and load another.

The existing saved-drawing path is a useful boundary: `SavedDrawingViews`
validates a presentation stream against its recipe, `FileWorkspace` retains
it, and `ParameterPackageTarget` publishes it with the save. An assembly needs
the graph-aware provenance and snapshot staging above in addition. The current
`FileContext::open_path` resolves assembly references eagerly; Express requires a
separate host load path that does not call that full reference resolution before
its cached scene is displayed. `DesktopPartScenes::evaluate_assembly` currently
owns traversal and presentation in the head crate, while save routes live in
the automation and file-workspace layers. Extract one shared assembly
presentation service and inject it into the common save path; do not copy the
renderer or cache only GUI saves. Viewport triangles already exist; no external
kernel capability has been identified as necessary for this cache.

The pinned scenario also expects enabled Finish and Simplify controls after
Load Full. The current assembly ribbon has no implemented Finish command, and
its Simplify command is muted. Those operations need genuine host behavior;
showing enabled placeholders would not satisfy the scenario.

## Required verification

Test cache round trips and rejection of missing, malformed, non-finite, stale,
reordered, cyclic, dirty and missing-dependency inputs, including nested and
repeated component references. Test failed full loading without changing the
active tab, saved paths, Express state or displayed geometry. Test successful
full loading with exact analytic bodies and no stale cache in modelling paths.
Test public typed client, lifecycle, existing-open mode conflicts, save/reopen,
and keyboard and mouse access. Run the pinned native scenario with a persisted
assembly whose cache is produced by the real save pipeline. Assert the prior
drawing tab, visible component, Load Full, full-mode controls and mode change.
