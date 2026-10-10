# ADR-0008: Vulkan renderer

Date: 2026-10-10. Status: Accepted for the Rust target design.
Implementation evidence: pending R1V/R4/R7 and Martensite capability checks.

Boundary: application presentation to Martensite visualization and Vulkan.
Owner: application viewport adapter; RustyCADKernel owns rendering algorithms
and GPU implementation. This is the one current application renderer ADR.

## Context

The Go records divide rendering across platform, style, lighting, instancing,
and OpenPBR decisions. The Rust port needs one coherent interface and resource
lifecycle. Martensite has a tested initial offscreen path; that does not yet
provide the full Oblikovati desktop or its required visual behavior.

## Decision

Use Martensite visualization and `martensite-vulkan` as the production renderer.
The viewport translates evaluated model results, cameras, appearances, lights,
selection, and overlays into immutable presentation inputs. Martensite does not
reach into application documents or load their resources through hidden I/O.
The shell uses a typed viewport interface, with no Vulkan handles in model code.

Keep a recording path for CPU presentation tests and an offscreen Vulkan path
for captures through the same presentation contract. They do not constitute
additional production modeling or shading engines. Small CPU reference
calculations exist only as independent rendering oracles.

## Interfaces and invariants

### Presentation and resource ownership

Martensite owns the Vulkan device, queues, view targets, command submission,
and GPU completion. The platform supplies the native window/surface lifetime.
The UI binding composes Dear ImGui through the same device/submission protocol;
it must not create an independent swapchain owner. R1V must establish the exact
Rust/native interface before application UI features depend on it.

Use explicit ownership for views, textures, descriptors, scene buffers, and
captures. Retire resources only after the submission that uses them completes,
including replaced icon textures and images exposed to ImGui. A fixed number
of elapsed frames is not proof of completion. Resize, minimized windows,
out-of-date surfaces, device errors, view removal, and shutdown all have
defined states. Stop submissions and wait for required completion before
destroying the view, device, and native window in their dependency order.

Captures identify their view, dimensions, scene/camera generation, and result
status. A failed redraw cannot return an old image as a fresh success. Input,
viewer mutation, and UI calls stay on their required owner threads; worker
tessellation can transfer results only under proved ownership rules.

### Capabilities and platforms

Use Vulkan 1.3 as the target backend. Query required features and extensions
and report the selected device/profile. Do not infer support for descriptor
indexing, device addresses, ray tracing, or presentation from a version number.
An optional path needs a tested equivalent or an explicit unavailable result.

Linux and Windows use their Vulkan runtime profiles. macOS uses a validated
MoltenVK profile with portability enumeration/subset handling and OS main-thread
rules. Old claims about which Vulkan features MoltenVK supplies are references,
not proof for the pinned Rust dependency. Packaging belongs to
[ADR-0013](ADR-0013-packaging-and-release.md).

Keep capability state in Martensite's renderer ledger and the application
behavior inventory. At the supplemental revision in the
[migration evidence](legacy-map.md#source-baseline), native window presentation,
highlight rendering, and much of the AIS presentation catalog remain open.
Required behavior beyond the OCCT renderer is a Martensite extension with
separate acceptance. No second application renderer fills these gaps.

### Display, light, and appearance behavior

Map the API display enum once to presentation configuration. Retain wireframe,
shaded, edge/hidden-edge, and stylized modes. One shared pass framework owns
common edge and stylization work. Public IDs and aliases come from the API.

Viewport hidden-edge display can use mesh depth for interactive presentation.
Drawing HLR, sections, dimensions, and exports consume analytic,
reference-bearing kernel curves under
[ADR-0007](ADR-0007-martensite-integration.md). A visually correct mesh cannot
certify drawing geometry or become its associative source.

Retain lighting presets, environment orientation/exposure, reflections, ground
and object shadows, ambient effects, and the visible background. A preset and
its defaults have one definition. Shaded and stylized surfaces respond to the
selected light rig; wire-only styles have no surface to shade. Resource
decoding is injected; GPU environment preparation belongs in Martensite.

The Realistic target is progressive OpenPBR Surface rendering, retaining the
legacy 1.1.1 requirement as a versioned compatibility target. Use one BSDF and
integrator with hardware and compute intersection backends. Hardware ray tracing
is optional: turning it off changes performance, not the supported material
model. Never label a reduced raster shader as fully compliant OpenPBR.
Reset accumulation on relevant scene, camera, light, and material revisions;
an explicitly identified interaction preview must not be mistaken for the
converged result. Validate every required lobe against pinned reference data.
Do not infer that all planned OpenPBR work shipped in the Go baseline.

Appearance state and old/new appearance compatibility belong to
[ADR-0010](ADR-0010-materials-and-appearances.md). The renderer consumes their
resolved values. Public parameters remain in Oblikovati.API.

### Instancing and picking

Share tessellation and GPU geometry by stable source geometry identity and
revision. Occurrences carry their own transforms, visibility, appearance
overrides, and selection identity. An assembly machining result with different
geometry has a different source revision; it cannot reuse an unchanged mesh.
Handle transform orientation and normals correctly for supported scale/mirror
operations. Pointer identity is not a persistent cache or occurrence key.

Pick buffers are temporary lookup aids. Each ID maps through presentation
provenance to an entity and occurrence in the displayed generation. Discard
stale picks. The model verifies the analytic reference before a command uses
it; no entity is named by a nearest triangle or tessellation order. Tool filter
and highlight semantics belong to [ADR-0009](ADR-0009-ui-and-selection.md).

## Alternatives and consequences

Porting the Go native renderer beside Martensite would duplicate resource and
shading ownership. Separate ADRs for styles or platform exceptions would split
the same presentation contract again. Reject both. Extending Martensite can
block UI milestones, so native presentation and shared ImGui composition are
early R1V proofs, not work left until release.

## Legacy disposition

Consolidate Oblikovati ADR-0005, 0014, runtime 0019, 0023, 0026, GPU 0038,
and rendering 0053, with renderer-facing 0004/0008 and the HLR distinction in
0012, at the [fixed revision](legacy-map.md#source-baseline). Replace the
old app-owned renderer and software-raster-only Realistic decision. Retain
mode behavior and oracle requirements; omit old descriptor layouts, magic
sample counts, fixed-frame retirement, and claims of zero regression risk.

## Validation required

R1V proves native presentation, ImGui composition, resize, capture, removal,
and shutdown using the real renderer. R4 verifies face picking and a modeled
shape after edits. R7 tests every required mode, overlays/text, lighting,
instancing, and Realistic with hardware RT on and off where available.

Require Vulkan validation with no errors, named color/depth/normal/ID outputs
where used, analytic/CPU checks, and transformation/instancing equivalence.
Use pinned software Vulkan and native target hardware; compare GPU images
with defined numeric/perceptual bounds, not universal cross-driver byte
identity. Blender reference scenes supplement these checks. Keep captures,
error regions, device details, and dependency revisions with live MCP evidence.
