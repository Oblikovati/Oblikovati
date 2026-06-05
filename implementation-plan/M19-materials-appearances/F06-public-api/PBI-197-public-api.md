# PBI-197 — Materials/appearances public API (wire + client + router)

> **Backfilled from shipped code 2026-06-04** (REPORT.md D-03). Grade: **M✅ · G n/a · U n/a**.

## Goal

Add-ins can list/get/create/update appearances & materials, assign, and read physics.

## API contracts

- `api/wire`: `appearances.{list,get,create,update}`, `materials.{list,get,create,update}`,
  `model.{assignMaterial,assignAppearance,physicalProperties}` (+ `AssetRefArgs` DTO).
- `api/contract.{Appearance,Material}`; typed `api/client` group.

## Scope / work

`addin/router` handlers keyed on the wire constants; dogfood client suite.

## Acceptance criteria

- Dogfood: create appearance → assign → physicalProperties returns mass (router test).
- `var _ contract.X` assertions hold.

## Depends on

PBI-192..196, ADR-0018.
