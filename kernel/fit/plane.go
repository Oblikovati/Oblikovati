// SPDX-License-Identifier: GPL-2.0-only

// Package fit derives geometry from sampled points. Its first tool is a least-squares best-fit
// plane (principal-component analysis of the points' covariance), used to turn a region of a
// scanned point cloud into a work plane the design can be built against (#645).
package fit

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// minFitPoints is the fewest points that define a plane.
const minFitPoints = 3

// Plane returns the least-squares best-fit plane through the points: its origin is their centroid
// and its normal is the direction of least variance — the eigenvector of the 3×3 covariance matrix
// with the smallest eigenvalue. It errors with fewer than three points, or when the points are
// collinear (or coincident), since those span no unique plane.
func Plane(points []math.Point3) (geom.Plane, error) {
	if len(points) < minFitPoints {
		return geom.Plane{}, fmt.Errorf("fit: need at least %d points to fit a plane, got %d", minFitPoints, len(points))
	}
	c := centroid(points)
	cov := covariance(points, c)
	vals, vecs := jacobiEigen3(cov)
	lo, mid := twoSmallest(vals)
	if vals[mid] <= planarityEps*vals[largestIndex(vals)] {
		return geom.Plane{}, fmt.Errorf("fit: points are collinear or coincident; no unique plane (eigenvalues %v)", vals)
	}
	normal := math.V3(math.Scalar(vecs[0][lo]), math.Scalar(vecs[1][lo]), math.Scalar(vecs[2][lo]))
	return geom.NewPlane(c, normal)
}

// planarityEps is the ratio below which the middle eigenvalue is treated as zero — the points
// extend in only one direction (a line), so no plane is determined.
const planarityEps = 1e-9

func centroid(points []math.Point3) math.Point3 {
	var sx, sy, sz float64
	for _, p := range points {
		sx, sy, sz = sx+float64(p.X), sy+float64(p.Y), sz+float64(p.Z)
	}
	n := float64(len(points))
	return math.P3(math.Scalar(sx/n), math.Scalar(sy/n), math.Scalar(sz/n))
}

// covariance accumulates the symmetric 3×3 scatter matrix of the centred points.
func covariance(points []math.Point3, c math.Point3) mat3 {
	var m mat3
	for _, p := range points {
		dx, dy, dz := float64(p.X-c.X), float64(p.Y-c.Y), float64(p.Z-c.Z)
		m[0][0] += dx * dx
		m[1][1] += dy * dy
		m[2][2] += dz * dz
		m[0][1] += dx * dy
		m[0][2] += dx * dz
		m[1][2] += dy * dz
	}
	m[1][0], m[2][0], m[2][1] = m[0][1], m[0][2], m[1][2] // symmetric
	return m
}

// twoSmallest returns the indices of the smallest and middle eigenvalues.
func twoSmallest(v [3]float64) (lo, mid int) {
	order := [3]int{0, 1, 2}
	for i := 0; i < 3; i++ {
		for j := i + 1; j < 3; j++ {
			if v[order[j]] < v[order[i]] {
				order[i], order[j] = order[j], order[i]
			}
		}
	}
	return order[0], order[1]
}

func largestIndex(v [3]float64) int {
	idx := 0
	for i := 1; i < 3; i++ {
		if v[i] > v[idx] {
			idx = i
		}
	}
	return idx
}
