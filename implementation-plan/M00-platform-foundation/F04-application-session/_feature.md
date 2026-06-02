---
milestone: M00
feature: F04
name: Application Session & Lifecycle
status: planned
---

# M00 · F04 — Application Session & Lifecycle

The top-level `Application` object that owns global services and the `Documents` collection, plus a lightweight `ApprenticeServer`/headless root over the same object model, version/locale info, and the options/preferences scaffold.

## In scope

- `Application` root and service directory (no logic god-object).
- `ApprenticeServer` headless/read-only mode.
- `SoftwareVersion`, locale, `UserName`.
- Options objects scaffold (`GeneralOptions`, etc.).

## Out of scope

_None._

## Key API contracts delivered

- `Application`,`_Application`
- `ApprenticeServer`,`ApprenticeServerComponent`
- `SoftwareVersion`
- `GeneralOptions`/`*Options`

## Depends on

F01-F03.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-011](PBI-011-application-root.md) | Application root object & service directory |
| [PBI-012](PBI-012-apprentice-headless.md) | ApprenticeServer headless/read-only root |
| [PBI-013](PBI-013-version-locale-options.md) | SoftwareVersion, locale & options scaffold |
