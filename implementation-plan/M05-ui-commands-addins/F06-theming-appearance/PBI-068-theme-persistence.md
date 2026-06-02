---
milestone: M05
feature: F06
pbi: PBI-068
title: Custom theme persistence (user config dir)
status: done
estimate: S
---

# PBI-068 — Custom theme persistence (user config dir)

**Milestone:** M05 · **Feature:** F06 UI Theming & Appearance

## Goal

Load and save custom themes and the selected-theme preference under the per-user config
directory, as readable YAML.

## Scope / work

- `theme/store.go`: `Store` over a `FileSystem` seam; `Load` (customs + active),
  `SaveTheme`, `RemoveTheme`, `SaveActive`; `DefaultRoot` via `os.UserConfigDir`; YAML via
  `persistence/yamlcodec`.
- `theme/os_fs.go`: `OSFileSystem` (creates dirs on write; tolerant of a missing dir/file).

## API contracts

- None (internal); reuses `persistence/yamlcodec`.

## Acceptance criteria

- Save→Load round-trips a custom theme's name/kind/colors; first run (no dir) loads empty;
  built-ins are refused for saving; theme files are human-readable YAML. Tested with an
  in-memory fake FS (no real disk IO).

## Depends on

PBI-067, ADR-0020.
