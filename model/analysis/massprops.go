// SPDX-License-Identifier: GPL-2.0-only

// Package analysis computes engineering analysis on the model (M18, #423): mass properties,
// measurement and (later) interference, FEA and simulation. It is a thin model-layer wrapper over
// the kernel's geometry computations, expressed in user units (millimetres / grams).
package analysis

import (
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// cmToMM converts the kernel's database length unit (centimetre) to the millimetres the analysis
// surface reports in. See [[database-unit-is-cm]].
const cmToMM = 10.0

// MassProperties is a part's combined geometry properties and mass, in user units.
type MassProperties struct {
	VolumeMm3      float64
	SurfaceAreaMm2 float64
	MassG          float64
	DensityGCm3    float64
	CentroidXMm    float64
	CentroidYMm    float64
	CentroidZMm    float64
}

// MassPropertiesOf returns the combined mass properties of the given solid bodies for a material
// density (g/cm³; ≤0 ⇒ 1.0, so the mass equals the volume in cm³). Volume, area and centroid come
// from each body's tessellated geometry (divergence theorem); the centroid is volume-weighted
// across the bodies.
func MassPropertiesOf(bodies []*topo.Body, densityGCm3 float64) MassProperties {
	if densityGCm3 <= 0 {
		densityGCm3 = 1
	}
	var volCm3, areaCm2, cx, cy, cz float64
	for _, b := range bodies {
		p := ops.BodyGeometryProperties(b, ops.DefaultQuality())
		volCm3 += p.Volume
		areaCm2 += p.Area
		cx += float64(p.Centroid.X) * p.Volume
		cy += float64(p.Centroid.Y) * p.Volume
		cz += float64(p.Centroid.Z) * p.Volume
	}
	if volCm3 > 0 {
		cx, cy, cz = cx/volCm3, cy/volCm3, cz/volCm3
	}
	return MassProperties{
		VolumeMm3:      volCm3 * cmToMM * cmToMM * cmToMM,
		SurfaceAreaMm2: areaCm2 * cmToMM * cmToMM,
		MassG:          densityGCm3 * volCm3,
		DensityGCm3:    densityGCm3,
		CentroidXMm:    cx * cmToMM,
		CentroidYMm:    cy * cmToMM,
		CentroidZMm:    cz * cmToMM,
	}
}
