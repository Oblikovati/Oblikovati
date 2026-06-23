// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/math"
)

// Serialized form of the network surface feature (M36-F10): the U- and V-direction curve polylines
// as nested [x,y,z] coordinate lists.

// NetworkSurfaceData is a network's recipe: the baked U/V curve polylines.
type NetworkSurfaceData struct {
	UCurves [][][]float64 `yaml:"uCurves"`
	VCurves [][][]float64 `yaml:"vCurves"`
}

// serializeNetworkSurface captures a network definition.
func serializeNetworkSurface(def *NetworkDefinition) *NetworkSurfaceData {
	return &NetworkSurfaceData{UCurves: encodePolylines(def.UCurves), VCurves: encodePolylines(def.VCurves)}
}

// restoreNetworkSurface rebuilds a network feature from its recipe.
func restoreNetworkSurface(fs *PartFeatures, d *NetworkSurfaceData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("network-surface feature is missing its payload")
	}
	return NewNetworkFeatures(fs).Add(decodePolylines(d.UCurves), decodePolylines(d.VCurves)), nil
}

// encodePolylines flattens curve polylines to nested [x,y,z] lists.
func encodePolylines(curves [][]math.Point3) [][][]float64 {
	out := make([][][]float64, len(curves))
	for i, c := range curves {
		out[i] = make([][]float64, len(c))
		for j, p := range c {
			out[i][j] = []float64{float64(p.X), float64(p.Y), float64(p.Z)}
		}
	}
	return out
}

// decodePolylines rebuilds curve polylines from nested [x,y,z] lists (skipping malformed points).
func decodePolylines(curves [][][]float64) [][]math.Point3 {
	out := make([][]math.Point3, len(curves))
	for i, c := range curves {
		out[i] = make([]math.Point3, 0, len(c))
		for _, p := range c {
			if len(p) == 3 {
				out[i] = append(out[i], math.P3(math.Scalar(p[0]), math.Scalar(p[1]), math.Scalar(p[2])))
			}
		}
	}
	return out
}
