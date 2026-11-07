# ADR-0036: One released Martensite model backend

Status: proposed; qualification in progress.
Date: 2026-11-07.

## Context

The host's native renderer and planar geometry already use Martensite. Its
B-rep primitives, arrangement diagnostic, validation, measurement, copy and
mesh boundaries still link the former RustyCAD v0.2.7 release. This leaves two
kernel type namespaces in one host and gives a Martensite renderer upgrade a
wider name than the evidence supports.

Martensite v0.13.0 publishes the corresponding primitive, Boolean and topology
packages. The Go application remains the architecture and implementation
reference. MCAD-Scenarios defines the host behavior to preserve.

## Decision

Replace every RustyCAD dependency and import with the released Martensite
v0.13.0 source, commit `d051ff699982faba4ec2df3ee9a47e14f38dedbd`.
Keep the existing project-owned adapter interfaces. Do not retain a second
backend, a runtime fallback, or an ordered try-list.

This deletes the RustyCAD package graph and its type namespace from the host.
It supersedes only the dependency-source choice recorded in earlier integration
records; their geometry, history, naming and validation requirements remain.
No model algorithm may depend on tessellation during the replacement.

Release verification must support the kernel's independently versioned crates.
It verifies the stable release's package catalog, its exact source-workspace
revision and checksum, and the versions consumed by Cargo. It must not invent
`0.13.0` versions for component packages or silently admit an unlisted package.
The Apache public API is unchanged by a private backend replacement.

## Qualification and consequences

The existing analytic prism corpus, per-face OCCT checks, copy isolation,
source lineage, reference-key round trips, edit/replay, native scene and
persistence checks are hard assertions against the replacement. A compile-only
or total-volume-only result does not establish compatibility.

An upstream capability gap returns its existing named decline and remains a
reported blocker. In particular, package publication alone does not close the
production sketch-region contract tracked in Martensite issue 390.
No recognizers, tolerance constants, fallback sites or geometry type switches
are added to make individual inputs pass.

The replacement is complete only after the old graph is absent and the
required local gate and native checks pass. Until then this record states an
implementation decision, not a completed migration or release claim.
