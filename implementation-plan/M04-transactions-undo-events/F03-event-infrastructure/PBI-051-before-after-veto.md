---
milestone: M04
feature: F03
pbi: PBI-051
title: Before/after timing & HandlingCode veto
status: planned
estimate: M
---

# PBI-051 — Before/after timing & HandlingCode veto

**Milestone:** M04 Transactions, Undo & Events  ·  **Feature:** F03 Event Infrastructure

## Goal

Implement two-phase events (before/after) and the veto mechanism via an out `HandlingCode`, plus a `NameValueMap` context for forward compatibility.

## Scope / work

- `EventTimingEnum` dispatch.
- `HandlingCodeEnum` (handled/not-handled/abort).
- Context bag passed to handlers.

## API contracts (interfaces / enums / collections)

- `EventTimingEnum`,`HandlingCodeEnum`,`NameValueMap`

## Acceptance criteria

- A before-handler returning abort cancels the operation.
- After-handlers see the committed result.

## Depends on

_See feature dependencies._
