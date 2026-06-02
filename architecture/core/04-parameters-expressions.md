# Core 04 — Parameters & expression engine

*Modernizes M02 (units, parameters, expressions, dependency graph). This layer is
almost unchanged in concept — it was already correct — but becomes much cleaner in
Go than in the variant-typed COM original.*

The parametric-cad skill (§4) is emphatic: dimensioned parameters + the dependency
DAG are ruinous to retrofit, so they come first. This is pure Go, cgo-free, and has
no dependency on the kernel or renderer — buildable and testable immediately.

## Quantities & units

Every numeric value is a **dimensioned quantity stored in database units** (cm,
radians) — the kernel never sees user units (parametric-cad §4a).

```go
package param
type Unit uint16    // length, angle, area, mass, unitless, bool, text (a stable TypeID-style enum)
type Quantity struct {
    Value float64    // ALWAYS in database units
    Unit  Unit
}
```

The COM `object Value` / `string Units` pair (variant + stringly-typed unit)
becomes a typed `Quantity`. Conversion/formatting happens only at the display and
parse boundary (`units.Format(q, userPrefs)`, `units.Parse("25 mm")`).

## The expression engine

A unit-aware parser/evaluator (`param/expr`): `"2 * width + 5 mm"` → AST →
dimensionally-checked evaluation. Pure Go, no variants:

```go
type Expr struct{ src string; ast node; refs []ParamID }   // refs by STABLE ID, not name
func (e Expr) Eval(env *Env) (Quantity, error)             // dimensional analysis enforced
```

- References bind by **stable `ParamID`**, never by display text (parametric-cad
  §4c) — so renaming a parameter is a label change, not a global string rewrite.
- Dimensional analysis rejects `1 mm + 1 deg` at eval time; the error becomes the
  parameter's **health**, never a panic.

## The parameter model

```go
type Parameter struct {
    id         ParamID
    Name       string          // display label (rename = relabel; refs unaffected)
    Expr       Expr            // authored source of truth
    value      Quantity        // evaluated, database units (cache)
    modelValue float64         // after tolerance (what the model consumes)
    Unit       Unit
    Tol        Tolerance
    Kind       Kind            // model | user | reference | derived | table
    Health     Health
}
```

The COM triad (`Expression` → `Value` → `ModelValue`) is preserved exactly — it was
right. `IsKey`, `Comment`, `ExposedAsProperty`, `Precision`, `DisplayFormat` carry
over as typed fields instead of variant properties.

## The dependency DAG (shared with the feature engine)

Parameters + expressions form a directed acyclic graph; this is the **same dirty-
propagation machinery** the scene-graph transforms (realtime-3d §3) and the feature
history (parametric-cad §2) use — one pattern, three users.

```go
type Graph struct{ nodes map[ParamID]*Parameter; edges adjacency }
func (g *Graph) Set(id ParamID, e Expr) error        // re-parse → update edges → detect cycles
func (g *Graph) DirtyClosure(id ParamID) []ParamID    // transitive dependents, topo-ordered
func (g *Graph) Recompute(ctx, dirty []ParamID) error // evaluate the affected sub-DAG only
```

- **Cycle detection on edit** → reject and mark sick (no crash).
- `DirtyClosure` returns the dependents in dependency order; only they recompute.
- The feature engine (iteration 2) hangs **features** off the same graph as
  dependents of the parameters they consume, so a parameter change dirties the
  right features automatically — and that recompute runs **async** (ADR-0007).

## Why Go makes this nicer than COM

| COM | Go |
|---|---|
| `object Value` (variant) | typed `Quantity` |
| `string Units` set from `UnitsTypeEnum` | typed `Unit` |
| references by expression text | references by stable `ParamID` |
| `HealthStatusEnum` checked via `Type` | typed `Health` field |
| `ExpressionList` COM collection | `[]Expr` / `iter.Seq[Expr]` |

Same domain semantics; the variant/stringly-typed surface that existed only to
cross COM is gone, replaced by the compiler checking units and references.
