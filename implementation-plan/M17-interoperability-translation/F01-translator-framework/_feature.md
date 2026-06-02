---
milestone: M17
feature: F01
name: Translator Add-in Framework
status: planned
---

# M17 · F01 — Translator Add-in Framework

The framework that makes import/export formats pluggable: the translator add-in contract, the translation context (source/target/data medium), capability queries, and `NameValueMap`-driven options.

## In scope

- `TranslatorAddIn`; open/save capability.
- `TranslationContext`; `DataMedium`.
- Options via `NameValueMap`.

## Out of scope

_None._

## Key API contracts delivered

- `TranslatorAddIn`,`TranslationContext`,`DataMedium`,`ApplicationAddInServer`(M05)

## Depends on

M05.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-158](PBI-158-translator-framework.md) | Translator add-in framework & context |
