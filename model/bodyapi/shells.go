// SPDX-License-Identifier: GPL-2.0-only

// Package bodyapi adapts kernel topology onto the api/contract body surface
// (M07-F05/F06/F07, Oblikovati/Oblikovati#628/#629/#630): face shells, wires,
// body queries and the transient B-rep factory.
package bodyapi

import (
	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Compile-time contract assertions (ADR-0018).
var (
	_ contract.FaceShell  = (*FaceShellAdapter)(nil)
	_ contract.FaceShells = (*FaceShellsAdapter)(nil)
	_ contract.Wire       = (*WireAdapter)(nil)
	_ contract.Wires      = (*WiresAdapter)(nil)
)

// onTolDefault is the boundary band of point containment when callers pass 0.
const onTolDefault = 1e-6

// FaceShellAdapter exposes one kernel shell as a contract FaceShell. It carries its owning body because
// void-ness is a property of the shell RELATIVE TO THE BODY (a lone shell is identical solid or void), so
// IsVoid classifies against the body rather than reading a tessellated signed volume (#3483).
type FaceShellAdapter struct {
	shell *topo.Shell
	body  *topo.Body
	q     ops.Quality
}

// NewFaceShell wraps a kernel shell of body b at the given quality.
func NewFaceShell(s *topo.Shell, b *topo.Body, q ops.Quality) *FaceShellAdapter {
	return &FaceShellAdapter{shell: s, body: b, q: q}
}

func (a *FaceShellAdapter) IsClosed() bool { return a.shell.IsClosed() }
func (a *FaceShellAdapter) IsVoid() bool   { return ops.ShellIsVoidInBody(a.body, a.shell) }

// Volume is the magnitude of the shell region's volume (the API reports
// sizes; the sign — void vs material — is IsVoid's job).
func (a *FaceShellAdapter) Volume() float64 {
	v := query.ShellSignedVolume(a.shell, a.q)
	if v < 0 {
		return -v
	}
	return v
}

func (a *FaceShellAdapter) FaceCount() int { return len(a.shell.Faces()) }
func (a *FaceShellAdapter) EdgeCount() int { return len(a.shell.Edges()) }

// IsPointInside classifies a point against the shell's bounded region.
func (a *FaceShellAdapter) IsPointInside(x, y, z float64) types.Containment {
	c := query.ShellContainment(a.shell, math.P3(math.Scalar(x), math.Scalar(y), math.Scalar(z)), a.q, onTolDefault)
	return containmentOf(c)
}

func (a *FaceShellAdapter) ReferenceKey() []byte { return a.shell.ReferenceKey() }
func (a *FaceShellAdapter) TransientKey() uint64 { return a.shell.ID() }

// containmentOf maps the kernel verdict onto the frozen wire enum.
func containmentOf(c query.PointContainment) types.Containment {
	switch c {
	case query.ContainInside:
		return types.InsideContainment
	case query.ContainOn:
		return types.OnContainment
	default:
		return types.OutsideContainment
	}
}

// FaceShellsAdapter enumerates a body's shells.
type FaceShellsAdapter struct {
	body *topo.Body
	q    ops.Quality
}

// NewFaceShells wraps a body's shell collection.
func NewFaceShells(b *topo.Body, q ops.Quality) *FaceShellsAdapter {
	return &FaceShellsAdapter{body: b, q: q}
}

func (a *FaceShellsAdapter) Count() int { return len(a.body.Shells()) }

func (a *FaceShellsAdapter) Item(index int) contract.FaceShell {
	return NewFaceShell(a.body.Shells()[index], a.body, a.q)
}

// WireAdapter exposes one kernel wire as a contract Wire.
type WireAdapter struct{ wire *topo.Wire }

// NewWire wraps a kernel wire.
func NewWire(w *topo.Wire) *WireAdapter { return &WireAdapter{wire: w} }

func (a *WireAdapter) IsClosed() bool       { return a.wire.IsClosed() }
func (a *WireAdapter) IsPlanar() bool       { return a.wire.IsPlanar() }
func (a *WireAdapter) EdgeCount() int       { return len(a.wire.Edges()) }
func (a *WireAdapter) ReferenceKey() []byte { return a.wire.ReferenceKey() }
func (a *WireAdapter) TransientKey() uint64 { return a.wire.ID() }

// WiresAdapter enumerates a body's wires.
type WiresAdapter struct{ body *topo.Body }

// NewWires wraps a body's wire collection.
func NewWires(b *topo.Body) *WiresAdapter { return &WiresAdapter{body: b} }

func (a *WiresAdapter) Count() int { return len(a.body.Wires()) }

func (a *WiresAdapter) Item(index int) contract.Wire {
	return NewWire(a.body.Wires()[index])
}
