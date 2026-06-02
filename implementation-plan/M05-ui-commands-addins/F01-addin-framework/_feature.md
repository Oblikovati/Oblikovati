---
milestone: M05
feature: F01
name: Add-in Framework
status: planned
---

# M05 · F01 — Add-in Framework

The extensibility entry point: the add-in server contract add-ins implement, the site the host provides, the registry of installed add-ins, and the automation object an add-in exposes.

## In scope

- `ApplicationAddInServer` lifecycle (Activate/Deactivate).
- `ApplicationAddInSite` host services.
- `ApplicationAddIns` registry; `AddInAutomation`.

## Out of scope

_None._

## Key API contracts delivered

- `ApplicationAddInServer`,`ApplicationAddInSite`,`ApplicationAddIns`,`ApplicationAddIn`,`AddInAutomation`

## Depends on

M04.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-054](PBI-054-addin-server-site.md) | Add-in server/site lifecycle & registration |
| [PBI-055](PBI-055-addin-automation.md) | Add-in automation object exposure |
