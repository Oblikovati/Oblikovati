---
milestone: M03
feature: F06
name: Attributes & Metadata
status: planned
---

# M03 · F06 — Attributes & Metadata

The generic metadata side-channel: typed named attributes any model object can carry (grouped in sets, keyed via reference keys so they survive recompute) and document-level iProperties/property sets that flow to BOMs and drawings.

## In scope

- `AttributeSet`/`Attribute` on any object.
- `PropertySets`/`PropertySet`/`Property` (iProperties).
- Parameter `ExposedAsProperty` bridge.

## Out of scope

_None._

## Key API contracts delivered

- `AttributeSets`,`AttributeSet`,`Attribute`,`ValueTypeEnum`
- `PropertySets`,`PropertySet`,`Property`

## Depends on

F05.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-044](PBI-044-attribute-sets.md) | AttributeSets/Attributes on any model object |
| [PBI-045](PBI-045-iproperties.md) | iProperties / PropertySets |
