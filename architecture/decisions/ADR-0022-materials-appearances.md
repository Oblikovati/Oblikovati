# ADR-0022 — Materials & Appearances (typed assets, project-scoped, PBR)

**Status:** accepted (user decision, 2026-06-02) · **Relates to:**
[ADR-0005](ADR-0005-vulkan13-renderer.md) (renderer), [ADR-0014](ADR-0014-renderer-testability.md)
(pure draw list), [ADR-0018](ADR-0018-apache-api-contract-module.md) (API split),
[ADR-0020](ADR-0020-yaml-git-friendly-document-format.md) (YAML documents),
[ADR-0021](ADR-0021-ui-theming-semantic-tokens.md) (reuses Rgba / the color picker).

## Context

Every body rendered with one hardcoded `renderer.defaultSurfaceColor` ("the neutral
material used until materials land"), bodies carried no physical properties, and there was
no way to author or assign look/material data. We want two linked subsystems: **Appearance**
(what the renderer shows) and **Material** (real-world physical properties), assignable to
bodies, with custom assets shared across a project's documents.

Inventor models these as generic **Asset** objects (a bag of typed `AssetValue`s); a
`MaterialAsset` references an appearance asset and a physical-properties asset; assets live
in project-scoped `AssetLibraries`; `AppearanceSourceTypeEnum` defines an override
precedence chain.

## Decision

1. **Typed structs, not a generic AssetValue bag.** `Appearance` (metallic-roughness PBR:
   albedo, metallic, roughness, emissive, opacity) and `Material` (density + Mechanical /
   Thermal / Electrical groups, referencing an appearance by id) are concrete typed Go
   structs in `model/material`, consistent with `model/param`/`model/sketch` and CLAUDE.md's
   "explicit types, no Dict". Inventor's names and the appearance-source precedence are kept.

2. **Solid PBR values now; no texture maps.** First milestone carries scalar/color PBR
   through `renderer.DrawItem`; the basic shader uses albedo and the rest feeds a future GGX
   upgrade. Image-texture maps are deferred (they need binary document sections + UVs).

3. **Three scope tiers, resolved document → project → built-in.** A shipped read-only
   catalog (built-ins), the active project's shared library (`DesignProject.Locations.
   DesignData`, `appearances.yaml`/`materials.yaml`), and per-document **embedded copies** of
   the non-built-in assets a document uses (so an `.obk` is self-contained). `MergedLookup`
   resolves embedded assets over the session catalog.

4. **Assignment by persistent reference key.** Assignments live on `PartComponentDefinition`
   keyed by `topo` `ReferenceKey` (stable across `Recompute`, unlike body id), with the
   precedence `face override → body appearance → body material → part material → part default
   → neutral default`.

5. **Public API (ADR-0018).** `types` (AssetSource, property groups, PhysicalProperties,
   reusing `Rgba`), `contract` (Appearance/Material), `wire` (`appearances.*`, `materials.*`,
   `model.assignMaterial`, `model.assignAppearance`, `model.physicalProperties`), `client`
   groups; served by `addin/router`.

6. **Renderer stays pure.** `BuildDrawList` takes a `SurfaceLookup` resolver built in the app
   from the active part's assignments + library; `renderer` never imports `model`.

7. **Mass properties from geometry.** A new `ops.BodyGeometryProperties` computes
   volume/area/centroid via the divergence-theorem signed-tetrahedron sum (oriented outward
   from per-vertex normals); mass = density × volume.

## Consequences

- New UI color/material data is added as typed fields + assets, not hardcoded colors.
- A document embeds the assets it uses, so it renders correctly without its project library;
  the project library is the shared catalog and the source of cross-document reuse.
- Body/face-level appearance overrides are modeled and resolved now; **face-level override
  rendering** (splitting a body mesh per face) and **image textures** are follow-on work.
- Live edit/assign updates the viewport next frame (the resolver reads the library each
  frame), matching the theming live-update model.
