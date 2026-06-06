// SPDX-License-Identifier: GPL-2.0-only

// Package feature is the feature-history recompute engine and the feature types
// built on it — the heart of part modeling (architecture modeling/01, ADR-0010).
// The model is an evaluated program: an ordered list of features, each a recipe
// (its Definition) that consumes the running body state plus its resolved inputs
// and produces new bodies (the Definition→Feature→geometry triangle, parametric-
// cad §3).
//
// Recompute is rollback-replay of the dirty tail (ADR-0010): on a change the engine
// finds the earliest dirty feature, reuses the cached body state from just before
// it, and replays forward to the end-of-part marker. A feature whose input
// reference is lost (or whose modeling op fails) goes [health.Sick] and poisons its
// dependents, never aborting the whole rebuild (parametric-cad §7). Feature inputs
// are reference keys ([Ref], resolved lazily against the current topology), not
// pointers, because the rollback-replay destroys and recreates topology every pass.
//
// Everything is pure and headless: PartFeatures.Recompute is a pure function of the
// feature list + parameters, fully unit-testable.
//
// Numeric definition fields are live values (func() float64), not frozen floats, so
// a parameter-driven dimension is re-read on every recompute (the recompute is what
// turns a parameter edit into new geometry). New features MUST follow this: store
// each dimensional argument as a func() float64 and call it inside Recompute. Where a
// constructor takes a plain float64 for convenience, wrap it with constFloat and also
// expose a func() variant for the parameter-aware caller (see AddThicken/AddThickenFn).
// The only fields that may stay plain floats are non-dimensional ones — solver
// tolerances and free-form sculpt sizes that are not meant to track parameters.
package feature
