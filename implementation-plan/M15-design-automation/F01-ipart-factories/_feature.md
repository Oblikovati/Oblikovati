---
milestone: M15
feature: F01
name: iPart/iAssembly Factories
status: planned
---

# M15 · F01 — iPart/iAssembly Factories

The factory model that turns a part/assembly into a family: a table of members (rows) with controlled columns (parameters, suppression, properties, components), generating member documents on demand with custom (per-instance) overrides.

## In scope

- `iPartFactory`/`iAssemblyFactory`; `iPartTableRow`/columns.
- Member generation & caching.
- Custom members; key columns.

## Out of scope

_None._

## Key API contracts delivered

- `iPartFactory`,`iPartTableRow(s)`,`iPartTableColumn(s)`,`iPartMember(s)`,`iAssemblyFactory`,`iAssemblyTableRow(s)`,`iFeature(s)`

## Depends on

M08.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-146](PBI-146-ipart-factory.md) | iPart/iAssembly factory & member generation |
| [PBI-147](PBI-147-ifeatures.md) | iFeatures (reusable feature templates) |
