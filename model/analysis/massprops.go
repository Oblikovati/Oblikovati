// SPDX-License-Identifier: GPL-2.0-only

// Package analysis computes engineering analysis on the model (M18, #423): mass properties,
// measurement and (later) interference, FEA and simulation. It is a thin model-layer wrapper over
// the kernel's geometry computations, expressed in user units (millimetres / grams).
package analysis

import (
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// cmToMM converts the kernel's database length unit (centimetre) to the millimetres the analysis
// surface reports in. See [[database-unit-is-cm]].
const cmToMM = 10.0

// MassProperties is a part's combined geometry properties, mass and inertia, in user units.
type MassProperties struct {
	VolumeMm3      float64
	SurfaceAreaMm2 float64
	MassG          float64
	DensityGCm3    float64
	CentroidXMm    float64
	CentroidYMm    float64
	CentroidZMm    float64
	// Mass moment of inertia about the centroid (g·mm²); Ixy/Iyz/Izx are products of inertia.
	InertiaXxGmm2 float64
	InertiaYyGmm2 float64
	InertiaZzGmm2 float64
	InertiaXyGmm2 float64
	InertiaYzGmm2 float64
	InertiaZxGmm2 float64
	// Principal moments of inertia (g·mm²) and their unit axes (rows aligned to the moments).
	PrincipalMomentsGmm2 [3]float64
	PrincipalAxes        [3][3]float64
}

// qualityFor maps an accuracy level to a tessellation quality for the FALLBACK path only.
//
// Mass properties integrate the ANALYTIC B-rep (ops.AnalyticGeometryProperties / AnalyticInertia,
// M48/C3 #3455/#3453/#3452), so for every body the analytic surface integrals set the accuracy and
// the result is exact for planar bodies and the analytic primitives — the old inscribed-N-gon
// deficit (a systematic −π²/(3N²) per curved feature, historically −0.64% against the Inventor
// oracle) is gone. A body the analytic path cannot yet cover (e.g. a trimmed NURBS whose uv
// boundary cannot be reconstructed) falls back to the tessellated integration at this quality,
// where the facet count still sets the accuracy: Low is a coarse interactive preview, Medium the
// parity-grade default, High the finest.
func qualityFor(accuracy types.MassPropertiesAccuracy) ops.Quality {
	switch accuracy {
	case types.MassPropertiesLow:
		return ops.Quality{ChordTolerance: 0.2, AngleTolerance: 20 * stdmath.Pi / 180}
	case types.MassPropertiesHigh:
		return ops.Quality{ChordTolerance: 1e-4, AngleTolerance: 0.5 * stdmath.Pi / 180}
	default:
		// Medium is the parity-grade default shared with the get_physical_properties readout.
		return ops.PropertyQuality()
	}
}

// MassPropertiesOf returns the combined mass properties of the given solid bodies for a material
// density (g/cm³; ≤0 ⇒ 1.0) at the given accuracy. Volume/area/centroid integrate each body's
// ANALYTIC B-rep (exact for planar bodies and the analytic primitives), falling back to the
// tessellated integration only for a body the analytic path cannot yet cover. The centroid is
// volume-weighted, and the inertia tensor is each body's inertia (about its own centroid) shifted
// to the combined centroid by the parallel-axis theorem.
func MassPropertiesOf(bodies []*topo.Body, densityGCm3 float64, accuracy types.MassPropertiesAccuracy) MassProperties {
	if densityGCm3 <= 0 {
		densityGCm3 = 1
	}
	q := qualityFor(accuracy)
	var volCm3, areaCm2, cx, cy, cz float64
	for _, b := range bodies {
		p := bodyGeometryProperties(b, q)
		volCm3 += p.Volume
		areaCm2 += p.Area
		cx += float64(p.Centroid.X) * p.Volume
		cy += float64(p.Centroid.Y) * p.Volume
		cz += float64(p.Centroid.Z) * p.Volume
	}
	if volCm3 > 0 {
		cx, cy, cz = cx/volCm3, cy/volCm3, cz/volCm3
	}
	mp := MassProperties{
		VolumeMm3:      volCm3 * cmToMM * cmToMM * cmToMM,
		SurfaceAreaMm2: areaCm2 * cmToMM * cmToMM,
		MassG:          densityGCm3 * volCm3,
		DensityGCm3:    densityGCm3,
		CentroidXMm:    cx * cmToMM,
		CentroidYMm:    cy * cmToMM,
		CentroidZMm:    cz * cmToMM,
	}
	applyInertia(&mp, bodies, q, densityGCm3, [3]float64{cx, cy, cz})
	return mp
}

// bodyGeometryProperties integrates a body's volume/area/centroid over its analytic B-rep, falling
// back to the tessellated path at quality q for a body the analytic path cannot yet cover (#3453).
func bodyGeometryProperties(b *topo.Body, q ops.Quality) ops.GeometryProperties {
	if p, ok := ops.AnalyticGeometryProperties(b); ok {
		return p
	}
	return ops.BodyGeometryProperties(b, q)
}

// bodyInertia integrates a body's inertia tensor over its analytic B-rep, falling back to the
// tessellated path at quality q for a body the analytic path cannot yet cover (#3452).
func bodyInertia(b *topo.Body, q ops.Quality) ops.InertiaTensor {
	if it, ok := ops.AnalyticInertia(b); ok {
		return it
	}
	return ops.BodyInertia(b, q)
}
