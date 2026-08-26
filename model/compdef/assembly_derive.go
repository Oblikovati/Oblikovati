// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"runtime"
	"sync"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/occurrence"
)

// An assembly definition is a body source a part can derive from: it flattens its
// occurrence tree into placed bodies (M11-F06). It already reports its geometry
// version, so a derived part re-derives when the assembly changes.
var _ feature.AssemblyBodySource = (*AssemblyComponentDefinition)(nil)

// PlacedBodies flattens the assembly's occurrence tree into the bodies of its leaf
// parts, each paired with the world transform that places it (composed down the
// occurrence path) and the occurrence it belongs to. Suppressed occurrences are
// skipped. It is what a derived-assembly component pulls to build a base body — and
// what shrinkwrap will simplify (M11-F06).
func (a *AssemblyComponentDefinition) PlacedBodies() []feature.PlacedBody {
	if a.occurrences.Count() >= parallelFlattenMinRoots {
		return flattenParallel(a.occurrences, a.WorkingScale())
	}
	return flattenSerial(a.occurrences, a.WorkingScale())
}

// parallelFlattenMinRoots is the top-level occurrence count at or above which PlacedBodies fans the
// flatten across workers; below it the goroutine/concat overhead outweighs the gain.
const parallelFlattenMinRoots = 4

// flattenSerial walks the whole tree on one goroutine — the small-assembly path and the reference
// the parallel path must match exactly.
func flattenSerial(occs *occurrence.Occurrences, rootWS float64) []feature.PlacedBody {
	w := newPlaceWalker()
	w.walk(occs, math.Identity4(), nil, rootWS)
	return w.out
}

// flattenParallel splits the top-level occurrences into contiguous chunks (one per worker, bounded
// by GOMAXPROCS) and walks each subtree concurrently, then concatenates the per-worker results in
// occurrence order. The walk is read-only over immutable topology and each worker owns its walker
// and output, so there is no shared mutable state; the contiguous, in-order concat makes the result
// byte-for-byte identical to flattenSerial. This is the M34-F2 wall-time cut over the deep DAG.
func flattenParallel(occs *occurrence.Occurrences, rootWS float64) []feature.PlacedBody {
	n := occs.Count()
	workers := min(runtime.GOMAXPROCS(0), n)
	parts := make([][]feature.PlacedBody, workers)
	var wg sync.WaitGroup
	for wkr := 0; wkr < workers; wkr++ {
		lo, hi := chunkRange(n, workers, wkr)
		wg.Add(1)
		go func(slot, lo, hi int) {
			defer wg.Done()
			w := newPlaceWalker()
			for i := lo; i < hi; i++ {
				w.place(occs.Item(i), math.Identity4(), nil, rootWS)
			}
			parts[slot] = w.out
		}(wkr, lo, hi)
	}
	wg.Wait()
	return concatPlaced(parts)
}

// chunkRange returns the half-open [lo, hi) of the n items assigned to worker w of workers,
// distributing the remainder to the lowest-numbered workers so every item is covered exactly once.
func chunkRange(n, workers, w int) (lo, hi int) {
	base, rem := n/workers, n%workers
	lo = w*base + min(w, rem)
	size := base
	if w < rem {
		size++
	}
	return lo, lo + size
}

// concatPlaced joins the per-worker outputs into one slice in worker order (which is occurrence
// order), preallocating the exact total so there is a single allocation.
func concatPlaced(parts [][]feature.PlacedBody) []feature.PlacedBody {
	total := 0
	for _, p := range parts {
		total += len(p)
	}
	out := make([]feature.PlacedBody, 0, total)
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// placeWalker carries the DFS state that used to be reallocated per node (M34-F2). The
// occurrence-name path is ONE stack pushed/popped down the tree and copied only when a leaf body
// is emitted (each PlacedBody keeps its own path), instead of a fresh growing slice per
// occurrence; iteration is by index (Occurrences.Item) so no per-level snapshot slice is
// allocated either. Output is byte-for-byte identical to the previous recursion — same DFS order,
// same paths — so this is a pure allocation cut, not a behaviour change.
type placeWalker struct {
	stack  []string // current occurrence-name path, reused across the whole walk
	out    []feature.PlacedBody
	active map[occurrence.Composite]bool // sub-assembly definitions on the current branch (cycle guard)
	depth  int                           // current recursion depth (depth guard)
}

// maxAssemblyDepth caps the occurrence-DAG recursion. Real assemblies are 6–8 levels (the
// automotive benchmark is 7); 256 is a generous backstop so a pathologically deep — or
// definition-cyclic — tree degrades to a bounded, finite flatten instead of overflowing the stack
// (M34-F6). The cycle set catches a definition reached through itself directly; the depth cap is the
// belt-and-braces limit for any other runaway nesting.
const maxAssemblyDepth = 256

// newPlaceWalker returns a walker with its cycle-guard set initialised.
func newPlaceWalker() placeWalker {
	return placeWalker{active: map[occurrence.Composite]bool{}}
}

// walk descends one occurrence level, composing each child's transform with parent's. flexParent,
// when non-nil, is the flexible sub-assembly occurrence whose children we are walking: each child
// uses that placement's independent override transform when one is set, so two placements of one
// flexible sub-assembly show components in different positions (M12-F06). The path disambiguates a
// shared flyweight reached through several placements (M11-F08).
func (w *placeWalker) walk(occs *occurrence.Occurrences, parent math.Matrix4, flexParent *occurrence.Occurrence, ownerWS float64) {
	for i := 0; i < occs.Count(); i++ {
		w.place(occs.Item(i), parent, flexParent, ownerWS)
	}
}

// place emits one occurrence's bodies (a leaf part) or recurses into it (a sub-assembly), at its
// world placement under parent. Suppressed occurrences are skipped. It pushes the occurrence name
// before descending and pops it after, so the shared stack always holds the current branch's path.
func (w *placeWalker) place(o *occurrence.Occurrence, parent math.Matrix4, flexParent *occurrence.Occurrence, ownerWS float64) {
	if o.Suppressed() {
		return
	}
	// Convert at the placement boundary (ADR-0042 Phase 2): the component's geometry is in its
	// own working unit; scale it into the owning assembly's working unit before placing. The
	// scale (childWS / ownerWS) is 1 when units match — every same-unit (and all-centimetre)
	// assembly is therefore unchanged.
	childWS := childWorkingScale(o.Definition(), ownerWS)
	world := parent.Mul(childTransform(o, flexParent).Mul(unitScale(childWS, ownerWS)))
	w.stack = append(w.stack, o.Name())
	switch def := o.Definition().(type) {
	case bodyDefinition: // a leaf part: emit its bodies placed in the assembly
		for _, b := range def.SurfaceBodies().All() {
			w.out = append(w.out, feature.PlacedBody{Body: b, Transform: world, Source: o, Path: w.clonePath()})
		}
	case occurrence.Composite: // a sub-assembly: recurse, its children converted from ITS working unit
		w.descend(def, o, world, childWS)
	}
	w.stack = w.stack[:len(w.stack)-1]
}

// childWorkingScale reports a definition's working scale (centimetres per working unit), or the
// owner's scale when the definition does not carry one (so no conversion is applied — patterns,
// test fakes). ADR-0042 Phase 2.
func childWorkingScale(def occurrence.Definition, ownerWS float64) float64 {
	if ws, ok := def.(interface{ WorkingScale() float64 }); ok {
		return ws.WorkingScale()
	}
	return ownerWS
}

// unitScale is the uniform similarity that converts a component's working-unit geometry into its
// owning assembly's working unit (childWS / ownerWS). It is the identity when the units match or
// either scale is degenerate, so the common path allocates the cheap identity matrix.
func unitScale(childWS, ownerWS float64) math.Matrix4 {
	if ownerWS <= 0 || childWS <= 0 || childWS == ownerWS {
		return math.Identity4()
	}
	s := math.Scalar(childWS / ownerWS)
	return math.Scale4(s, s, s)
}

// descend recurses into a sub-assembly under the cycle and depth guards (M34-F6): a definition
// already on the current branch (a self-containing assembly) or a branch past maxAssemblyDepth is
// skipped, so a malformed DAG degrades to a bounded flatten instead of overflowing the stack. The
// flexible-child override resets unless this occurrence is itself flexible.
func (w *placeWalker) descend(def occurrence.Composite, o *occurrence.Occurrence, world math.Matrix4, defWS float64) {
	if w.depth >= maxAssemblyDepth || w.active[def] {
		return
	}
	var nextFlex *occurrence.Occurrence // reset: a non-flexible sub-assembly clears the override
	if o.Flexible() {
		nextFlex = o
	}
	w.active[def] = true
	w.depth++
	// The sub-assembly's children are authored in ITS working unit, so it is their owner scale.
	w.walk(def.Occurrences(), world, nextFlex, defWS)
	w.depth--
	delete(w.active, def)
}

// clonePath returns an owned copy of the current name path; each PlacedBody retains its own, so a
// path never aliases the walker's mutating stack.
func (w *placeWalker) clonePath() occurrence.OccurrencePath {
	p := make(occurrence.OccurrencePath, len(w.stack))
	copy(p, w.stack)
	return p
}

// childTransform is o's local placement, replaced by flexParent's per-child override when this
// occurrence is a child of a flexible sub-assembly that pins it to an independent transform.
func childTransform(o *occurrence.Occurrence, flexParent *occurrence.Occurrence) math.Matrix4 {
	if flexParent != nil {
		if override, ok := flexParent.ChildTransform(o.Name()); ok {
			return override
		}
	}
	return o.Transform()
}

// bodyDefinition is a component definition that owns evaluated bodies — a part
// definition. Matched structurally so the flatten works for any body-bearing leaf.
type bodyDefinition interface {
	SurfaceBodies() *topo.SurfaceBodies
}
