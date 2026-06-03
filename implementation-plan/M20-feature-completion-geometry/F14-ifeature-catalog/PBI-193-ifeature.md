---
milestone: M20
feature: F14
pbi: PBI-193
title: iFeature extract & place
status: planned
estimate: M
---

# PBI-193 — iFeature extract & place

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F14 iFeature Catalog

## Goal

Capture selected features + their parameters/inputs as an `iFeature` definition, and place parameterized instances.

## Scope / work

`iFeatureDefinition` (captured features, exposed parameters, input placeholders); extract from a part; `iFeatures.Add` places at a location binding the input references and overriding exposed params; .obk + catalog file round-trip.

## API contracts (interfaces / enums / collections)

- `iFeature(s)`/`iFeatureDefinition`/`iFeatureParameter`; catalog read/write behind a thin interface.

## Acceptance criteria

- Extracting a hole+fillet pair to an iFeature then placing it on another face reproduces both features with overridden diameter
- round-trips.

## Depends on

M08, M20·F01

## Notes

Follows the canonical feature pattern set by Extrude (M08·PBI-092): full
`Definition → Add → Feature(+Proxy)` triangle, `/api` `wire`+`client`+`contract`,
`addin/router` handler, `.obk` round-trip, and executable tests.
