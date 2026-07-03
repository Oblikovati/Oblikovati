# head/ui consumer interfaces — policy (audit I5)

**Status:** active policy · **Applies to:** `head/ui` · **From:** 2026-07 deep audit, finding I5.

## The problem

`head/ui` is god-coupled to the concrete `*app.Session`: hundreds of references
across the package reach directly into the whole session. Every widget takes the
full session, so every widget *can* touch anything — business rules leak into
widgets, nothing documents what a widget actually needs, no widget is testable
against a slim fake host, and a `grep` for `*app.Session` cannot tell load-bearing
uses from incidental ones. This is the convention-only coupling behind the shipped
drift bugs (#1416, #1426, #1521).

## The policy — boy-scout, not big-bang

Every widget **touched from now on** declares its own **consumer-side interface**
of **≤6 methods** and takes that instead of `*app.Session`. `*app.Session`
satisfies it implicitly ("accept interfaces, return structs"; Interface Segregation
Principle). Existing untouched widgets are left alone — there is no big-bang
rewrite.

Templates already in the repo: `head/ui/viewcube_arrows.go` (`arrowSession`) and
`head/ui/viewport_cache.go` (`modelGeometryVersioned`). First tranche converted for
this note: `commitCancelHost` (the shared OK/Cancel row every tool dialog draws
through, `dialog_buttons.go`), `measureHost` and `gripSnapHost` (the Measure and
Grip Snap panels, `measure_dialog.go` / `grip_snap_dialog.go`), and
`activeDocumentSource` / `edgeColorSource` (the viewport-cache helpers,
`viewport_cache.go`).

### Recipe

1. Name the interface after the widget's role, not the session:
   `type browserSession interface { … }`. Keep it to ≤6 methods — the methods the
   widget actually calls.
2. Prove `*app.Session` satisfies it with a compile-time assertion beside the
   interface: `var _ browserSession = (*app.Session)(nil)`.
3. Take the interface as the parameter: `func drawBrowser(s browserSession)`.
   Where a widget is dispatched by a registry that stores `func(*app.Session)`
   (e.g. the tool-dialog registry, I4), keep a thin `func(s *app.Session)` adapter
   that delegates to the interface-typed body — the seam is the body.
4. Add a unit test driving the widget from a **named fake host** (CLAUDE.md: named
   fake classes, not inline stubs), e.g. `fakeMeasureHost`. Previously impossible.

## The ratchet

`archguard.TestHeadSessionCouplingRatchet` pins the count of `*app.Session`
references in `head/ui` production code (excluding doc-comment prose and the
`(*app.Session)(nil)` seam assertions). The number **only goes down**: a new
`*app.Session` parameter raises it and fails CI; a conversion lowers it and the pin
is lowered to match. This turns the "convert opportunistically" policy into a
mechanically enforced floor.
