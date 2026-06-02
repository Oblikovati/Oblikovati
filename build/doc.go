// SPDX-License-Identifier: GPL-2.0-only

// Package build exposes compile-time configuration and small cross-cutting
// helpers shared by every layer of Oblikovati.
//
// Mode flags (Debug, Profile, Editor) are plain constants selected by build
// tags so that dead branches compile out instead of being checked at runtime
// (see architecture/core/01-module-layout.md). Toggle them per build:
//
//	go build -tags debug,profile ./cmd/oblikovati
//
// Version metadata is injected at link time; see version.go.
package build
