// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// "Output is byte-identical across runs and platforms: ... FMA-safe arithmetic in predicates"
// (kernel ground rules) — enforced here for the whole arithmetic floor, per ADR-0064 (#3528).
//
// Go LICENSES fusion: "an implementation may combine multiple floating-point operations into a
// single fused operation, possibly ACROSS STATEMENTS". gc takes it on arm64 and never on amd64, so
// the same source computes different last bits on the macOS leg than on the Linux and Windows ones.
// Four decision flips found that way are fixed at their root in ADR-0061; this guard removes the
// difference itself rather than the decisions that could see it.
//
// The rule is stated on the PRODUCT, not on the sum, because every "the sum is written next to the
// product" formulation is unsound. Measured on go1.27.0 arm64, all of these fuse:
//
//	p := a * b; return p + c            // across statements
//	return box{a * b}.v + c            // through a struct field
//	return ident(a*b) + c              // through an inlined identity function
//	return Vector3{v.X * s}.X + c      // through an inlined constructor
//	return a + b/2                     // a power-of-two divide is strength-reduced to a multiply
//
// So: a floating-point multiplication or division must be explicitly rounded unless its value is
// consumed immediately by another multiplication or division (which cannot fuse it) or by a
// comparison (arm64 has no fused multiply-compare). The Go spec guarantees an explicit conversion
// rounds, which is what blocks the contraction; math.FMA everywhere was rejected in ADR-0064
// because it would move every stored fingerprint and every ADR oracle number on amd64.
var fmaPolicyPackages = []string{
	"oblikovati.org/math",
	"oblikovati.org/kernel/geom",
	"oblikovati.org/kernel/predicates",
}

// complexFusionDebt pins the sites this policy CANNOT fix: complex128 arithmetic. A conversion
// complex128(a*b) does NOT block the contraction (measured, ADR-0064), and complex division runs
// inside runtime.complex128div, which this repo does not compile. Ferrari's quartic factoring is
// the only complex path in the covered packages. It may only shrink — closing it means writing
// that solve in real arithmetic (#3528 follow-up).
var complexFusionDebt = map[string]int{
	"kernel/geom/quartic_real_roots.go": 5,
}

func TestNoFusableProductSums(t *testing.T) {
	t.Parallel()
	unrounded, complexes := scanFusableProducts(t)
	if len(unrounded) > 0 {
		t.Errorf("%d floating-point product(s)/quotient(s) can still be contracted into a following "+
			"add on arm64, so this package computes different last bits on the macOS CI leg than on "+
			"Linux and Windows. Round each at its SOURCE — float64(a*b), or Scalar(a*b) in "+
			"oblikovati.org/math — not at the sum (ADR-0064, #3528):\n  %s",
			len(unrounded), strings.Join(unrounded, "\n  "))
	}
	assertComplexDebtHolds(t, complexes)
}

// assertComplexDebtHolds ratchets the complex128 residual: it may fall (lower the pin) but never rise.
func assertComplexDebtHolds(t *testing.T, got map[string]int) {
	t.Helper()
	var rose, fell, stale []string
	for path, n := range got {
		switch owed := complexFusionDebt[path]; {
		case n > owed:
			rose = append(rose, path+": "+strconv.Itoa(n)+" complex site(s), budget "+strconv.Itoa(owed))
		case n < owed:
			fell = append(fell, `"`+path+`": `+strconv.Itoa(n)+",")
		}
	}
	for path := range complexFusionDebt {
		if _, ok := got[path]; !ok {
			stale = append(stale, path)
		}
	}
	sort.Strings(rose)
	sort.Strings(fell)
	sort.Strings(stale)
	reportComplexDebt(t, rose, fell, stale)
}

func reportComplexDebt(t *testing.T, rose, fell, stale []string) {
	t.Helper()
	if len(rose) > 0 {
		t.Errorf("new complex128 arithmetic in the FMA-policy packages — it CANNOT be made "+
			"platform-stable (complex128(x) does not round, and complex division is in the runtime). "+
			"Write it in real arithmetic instead (ADR-0064):\n  %s", strings.Join(rose, "\n  "))
	}
	if len(fell) > 0 {
		t.Errorf("complex-fusion debt FELL — good; lower these complexFusionDebt entries so the "+
			"ratchet holds the new floor:\n  %s", strings.Join(fell, "\n  "))
	}
	if len(stale) > 0 {
		t.Errorf("these complexFusionDebt files carry no complex arithmetic any more — DELETE their "+
			"entries:\n  %s", strings.Join(stale, "\n  "))
	}
}

// scanFusableProducts type-checks each covered package from source and returns the unrounded real
// sites (as "file:line: text") and the per-file count of unrounded complex ones.
func scanFusableProducts(t *testing.T) ([]string, map[string]int) {
	t.Helper()
	fset := token.NewFileSet()
	im := newModuleSourceImporter(fset, "..", filepath.Join("..", "..", "Oblikovati.API"))
	var unrounded []string
	complexes := map[string]int{}
	for _, path := range fmaPolicyPackages {
		dir, _ := im.dirFor(path)
		info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}}
		_, files, err := im.check(path, dir, info)
		if err != nil {
			t.Fatalf("type-checking %s from source failed, so this guard would pass vacuously: %v", path, err)
		}
		for _, f := range files {
			collectFusableProducts(fset, info, f, &unrounded, complexes)
		}
	}
	sort.Strings(unrounded)
	return unrounded, complexes
}

// collectFusableProducts appends every float product or quotient in f that the compiler may still
// contract, and tallies the complex ones per file. TWO node shapes carry one: a written `a * b`,
// and a compound assignment `x *= a` — which contains no BinaryExpr at all and is therefore
// invisible to a walk that only inspects expressions, while compiling to the same FMADDD.
func collectFusableProducts(fset *token.FileSet, info *types.Info, f *ast.File, unrounded *[]string, complexes map[string]int) {
	parents := parentLinks(f)
	ast.Inspect(f, func(n ast.Node) bool {
		switch e := n.(type) {
		case *ast.BinaryExpr:
			if e.Op == token.MUL || e.Op == token.QUO {
				recordFusable(fset, e, fusableKind(info, e), roundedByContext(info, parents, e), unrounded, complexes)
			}
		case *ast.AssignStmt:
			recordFusableAssign(fset, info, e, unrounded, complexes)
		}
		return true
	})
}

// recordFusableAssign reports a compound multiply- or divide-assignment on a floating-point
// left-hand side. `x *= a` IS the product `x * a`, and it cannot be wrapped where it is written:
// the rewrite is `x = float64(x * a)`.
func recordFusableAssign(fset *token.FileSet, info *types.Info, as *ast.AssignStmt, unrounded *[]string, complexes map[string]int) {
	if as.Tok != token.MUL_ASSIGN && as.Tok != token.QUO_ASSIGN {
		return
	}
	for _, lhs := range as.Lhs {
		recordFusable(fset, as, fusableKind(info, lhs), false, unrounded, complexes)
	}
}

// recordFusable files one site under its kind, unless its context already rounds it.
func recordFusable(fset *token.FileSet, n ast.Node, kind productKind, rounded bool, unrounded *[]string, complexes map[string]int) {
	if kind == notFusable || rounded {
		return
	}
	pos := fset.Position(n.Pos())
	rel := relativeToRepo(pos.Filename)
	if kind == complexProduct {
		complexes[rel]++
		return
	}
	*unrounded = append(*unrounded, rel+":"+strconv.Itoa(pos.Line)+": "+nodeText(fset, n))
}
