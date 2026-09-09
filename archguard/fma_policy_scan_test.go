// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
)

// productKind classifies a multiplication or division for the FMA policy (ADR-0064).
type productKind int

const (
	notFusable     productKind = iota // integer, or a constant the compiler folds
	realProduct                       // float32/float64 — fixable with an explicit conversion
	complexProduct                    // complex64/complex128 — NOT fixable; see complexFusionDebt
)

// fusableKind reports whether e is arithmetic the compiler may contract into a following add.
// A constant expression is folded at compile time and can never be.
func fusableKind(info *types.Info, e ast.Expr) productKind {
	tv, ok := info.Types[e]
	if !ok || tv.Type == nil || tv.Value != nil {
		return notFusable
	}
	b, ok := tv.Type.Underlying().(*types.Basic)
	if !ok {
		return notFusable
	}
	switch {
	case b.Info()&types.IsFloat != 0:
		return realProduct
	case b.Info()&types.IsComplex != 0:
		return complexProduct
	}
	return notFusable
}

// roundedByContext reports whether a product's value cannot reach an add unrounded: it is already
// inside an explicit conversion, or it feeds another multiplication, division or comparison.
func roundedByContext(info *types.Info, parents map[ast.Node]ast.Node, m ast.Node) bool {
	for {
		switch p := parents[m].(type) {
		case *ast.ParenExpr:
			m = p
		case *ast.UnaryExpr:
			if p.Op != token.SUB && p.Op != token.ADD {
				return false
			}
			m = p
		case *ast.BinaryExpr:
			return p.Op == token.MUL || p.Op == token.QUO || isComparison(p.Op)
		case *ast.CallExpr:
			tv, ok := info.Types[p.Fun]
			return ok && tv.IsType()
		default:
			return false
		}
	}
}

// isComparison reports whether op compares rather than computes. arm64 has no fused
// multiply-compare, so a product feeding a comparison cannot be contracted.
func isComparison(op token.Token) bool {
	switch op {
	case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
		return true
	}
	return false
}

// parentLinks maps every node in f to its syntactic parent.
func parentLinks(f *ast.File) map[ast.Node]ast.Node {
	parents := map[ast.Node]ast.Node{}
	var link func(n ast.Node)
	link = func(n ast.Node) {
		ast.Inspect(n, func(c ast.Node) bool {
			if c == nil || c == n {
				return true
			}
			parents[c] = n
			link(c)
			return false
		})
	}
	link(f)
	return parents
}

// exprText renders an expression back to source so the failure names the offending line.
func exprText(fset *token.FileSet, e ast.Expr) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, e); err != nil {
		return "<unprintable>"
	}
	return buf.String()
}

// relativeToRepo turns an absolute scan path into a repo-relative one for the debt table.
func relativeToRepo(path string) string {
	rel, err := filepath.Rel("..", path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return strings.TrimPrefix(filepath.ToSlash(rel), "./")
}
