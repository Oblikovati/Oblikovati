//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

// Shared UI literals: the import-failure notice format and the Dear ImGui id prefix
// that keeps tool-parameter widgets unique per label.
const (
	importFailedFmt = "Import failed: %v"
	toolParamPrefix = "##tool-param-"
)
