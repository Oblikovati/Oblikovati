// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
)

// The typed-handler adapters (M40 audit G1, #1649). Every wire handler used to hand-roll the
// same four-step scaffold — declare the args DTO, decode it, resolve the active model context,
// marshal the result — so a one-line change to error shaping or decode strictness was a
// hundreds-of-sites sweep and the copies drifted. These adapters capture the scaffold ONCE and
// let a handler shrink to its real behavior: a typed `func(*app.Session, Ctx, Args) (Result, error)`
// registered as `r.readOnly(wire.MethodX, typedPart(getX))`. The json.RawMessage→json.RawMessage
// seam the router dispatches on is unchanged; only the per-handler boilerplate inside it is gone.
//
// Canonical order is context-BEFORE-decode, matching the codebase's existing generics
// (decodeFeatureArgs, twoRefArgs, holderAndSetArgs): "is there even an active part?" is a
// precondition independent of the args, so it is checked first. Each individual failure keeps its
// exact error (the decode wrap, the modelaccess not-a-part text), so an add-in observes the same
// error for a decode failure and for no-active-context as before.

// marshalResult is the shared tail every adapter ends on: propagate a handler error unwrapped, or
// marshal its typed result to the wire. Centralising it means the decode→run→marshal error shaping
// lives in exactly one place.
func marshalResult[Result any](out Result, err error) (json.RawMessage, error) {
	if err != nil {
		return nil, err
	}
	return json.Marshal(out)
}

// typed adapts a context-free handler (decode Args, run, marshal Result) to the router's raw
// handlerFunc. Use it for methods that read no active model context — e.g. the unit/expression
// service: `r.readOnly(wire.MethodUnitsConvert, typed(unitsConvert))`.
func typed[Args, Result any](h func(*app.Session, Args) (Result, error)) handlerFunc {
	return func(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
		var in Args
		if err := decode(raw, &in); err != nil {
			return nil, err
		}
		return marshalResult(h(s, in))
	}
}

// typedCtx is the workhorse behind typedPart/typedAssembly/typedHolder: resolve a model context
// via resolve (part, assembly, parameter holder…), decode Args, run h, marshal Result. It is
// generic over the resolver's context type so the same scaffold serves every active-model kind.
func typedCtx[Ctx, Args, Result any](
	resolve func(*app.Session) (Ctx, error),
	h func(*app.Session, Ctx, Args) (Result, error),
) handlerFunc {
	return func(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
		ctx, err := resolve(s)
		if err != nil {
			return nil, err
		}
		var in Args
		if err := decode(raw, &in); err != nil {
			return nil, err
		}
		return marshalResult(h(s, ctx, in))
	}
}

// ctxQuery is typedCtx without arguments: resolve a model context, run h, marshal Result. It
// backs partQuery/assemblyQuery/holderQuery so no-argument list/get handlers do not have to
// declare an empty args DTO just to satisfy typedCtx.
func ctxQuery[Ctx, Result any](
	resolve func(*app.Session) (Ctx, error),
	h func(*app.Session, Ctx) (Result, error),
) handlerFunc {
	return func(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
		ctx, err := resolve(s)
		if err != nil {
			return nil, err
		}
		return marshalResult(h(s, ctx))
	}
}

// typedPart adapts a handler that needs the active part and typed args:
// `r.mutating(wire.MethodWorkSurfacesRename, "Rename Work Surface", typedPart(renameWorkSurface))`.
func typedPart[Args, Result any](
	h func(*app.Session, *compdef.PartComponentDefinition, Args) (Result, error),
) handlerFunc {
	return typedCtx(modelaccess.ActivePart, h)
}

// partQuery adapts a no-argument handler that needs the active part:
// `r.readOnly(wire.MethodWorkSurfacesList, partQuery(listWorkSurfaces))`.
func partQuery[Result any](
	h func(*app.Session, *compdef.PartComponentDefinition) (Result, error),
) handlerFunc {
	return ctxQuery(modelaccess.ActivePart, h)
}

// typedAssembly adapts a handler that needs the active assembly and typed args.
func typedAssembly[Args, Result any](
	h func(*app.Session, *compdef.AssemblyComponentDefinition, Args) (Result, error),
) handlerFunc {
	return typedCtx(modelaccess.ActiveAssembly, h)
}

// assemblyQuery adapts a no-argument handler that needs the active assembly.
func assemblyQuery[Result any](
	h func(*app.Session, *compdef.AssemblyComponentDefinition) (Result, error),
) handlerFunc {
	return ctxQuery(modelaccess.ActiveAssembly, h)
}

// typedHolder adapts a handler that needs the active parameter holder (part OR assembly) and args.
func typedHolder[Args, Result any](
	h func(*app.Session, compdef.ParameterHolder, Args) (Result, error),
) handlerFunc {
	return typedCtx(modelaccess.ActiveParameterHolder, h)
}

// holderQuery adapts a no-argument handler that needs the active parameter holder.
func holderQuery[Result any](
	h func(*app.Session, compdef.ParameterHolder) (Result, error),
) handlerFunc {
	return ctxQuery(modelaccess.ActiveParameterHolder, h)
}

// indexed is the Count()/Item(i) collection shape the model's typed collections expose (work
// surfaces, blocks, occurrences…). projectAll ranges over it.
type indexed[I any] interface {
	Count() int
	Item(int) I
}

// projectAll maps a Count()/Item(i) collection to a wire-row slice via project, replacing the
// hand-rolled `out := make([]Info, col.Count()); for i … { out[i] = project(i, col.Item(i)) }`
// index-projection loop the list handlers repeated (former G10). Example:
// `projectAll(part.WorkSurfaces(), workSurfaceInfo)`.
func projectAll[I, Info any](col indexed[I], project func(int, I) Info) []Info {
	out := make([]Info, col.Count())
	for i := range out {
		out[i] = project(i, col.Item(i))
	}
	return out
}
