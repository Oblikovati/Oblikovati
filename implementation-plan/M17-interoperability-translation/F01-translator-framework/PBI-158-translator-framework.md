---
milestone: M17
feature: F01
pbi: PBI-158
title: Translator add-in framework & context
status: planned
estimate: M
---

# PBI-158 — Translator add-in framework & context

**Milestone:** M17 Interoperability & Translation  ·  **Feature:** F01 Translator Add-in Framework

## Goal

Implement the translator framework so any format is a pluggable add-in exposing capabilities, a translation context, and an options bag.

## Scope / work

- `TranslatorAddIn` (HasOpen/HasSave, capabilities).
- `TranslationContext` + `DataMedium` (file/stream).
- Options schema via `NameValueMap`.

## API contracts (interfaces / enums / collections)

- `TranslatorAddIn`,`TranslationContext`,`DataMedium`

## Acceptance criteria

- A registered translator advertises capabilities and is invoked for its format with options.

## Depends on

_See feature dependencies._
