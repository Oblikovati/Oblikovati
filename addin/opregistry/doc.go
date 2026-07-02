// SPDX-License-Identifier: GPL-2.0-only

// Package opregistry is the self-describing registry of model-mutating operations
// the bridge exposes that have no first-party core registry yet (features and,
// later, sketch geometry). Each operation is an [OperationDescriptor]: a name, a
// one-line summary, a JSON Schema for its arguments (so an LLM can fill them), and
// an Apply func over the live session. Adding a feature kind to the MCP surface is
// "register one descriptor" — features.list and the schema resources read this
// registry, mirroring the future core registry (ADR-0003).
//
// Entry points: [New] builds an empty [Registry]; [Default] is the full built-in
// operation set the router serves. Apply funcs run on the session goroutine (via
// the Dispatcher), so they may touch the model directly.
//
// # Parameter-aware arguments (MANDATORY for every new operation)
//
// Every numeric argument of a feature or tool MUST be parameter-aware and survive
// parameter edits. Do not parse an argument to a frozen float64; resolve it through
// one of the *Closure helpers (lengthClosure, angleClosure, numberClosure, and
// optionalAngleClosure in feature_refs.go) and pass the returned func() value to the
// model builder. A plain literal ("10 mm") becomes a constant;
// any expression ("h", "od/2") is backed by an auto model parameter so it joins the
// dependency graph. set_parameter then marks features dirty and the next recompute
// re-reads the closure at the new value (see addin/router/parameters.go and
// compdef.PartComponentDefinition.Recompute). The model feature/tool field MUST be a
// func() float64 (not a float64) so the value is read live on every recompute.
//
// The only permitted exceptions, which take a literal value, are non-dimensional
// arguments: solver tolerances (stitch tolerance, mid-surface thin-wall threshold)
// and free-form primitive sizes (the subdivision cage is generated once at creation
// and then sculpted, so it has no live size). Integer-count arguments declared as
// JSON ints (pattern occurrences) are a known gap pending a string-expression DTO.
package opregistry
