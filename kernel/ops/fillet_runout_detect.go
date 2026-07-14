// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// runoutImprint is one host plane's coplanar feature footprint whose base curve crosses the
// receded fillet band twice and genuinely dips into it (the runout-imprint trigger, plan
// docs/superpowers/plans/2026-07-14-curved-runout-imprint-fillet.md). It carries everything a
// later imprint-solve/apply task needs: the host face and its plane frame, the closed footprint
// edge (the feature's inner-loop base curve), and the two band crossings already resolved in
// host-plane 2D.
type runoutImprint struct {
	host          *topo.Face // the fillet's planar face carrying the footprint
	hostIsA       bool
	plane         geom.Plane
	footprintEdge *topo.Edge // the closed feature base curve on host (inner loop)
	nodes         [2]crossing
	flat          func(math.Point3) math.Point2
	back          func(math.Point2) math.Point3
}

// detectRunouts finds, per host plane of a straight (constant-radius) plane∧plane fillet edge, a
// coplanar feature footprint crossing the receded band — the runout-imprint case a later task
// merges into the host loop and trims the fillet against, WITHOUT rebuilding a new blend surface
// (unlike the mid-span obstacle path, ADR-4). Unlike detectObstacle (fillet_obstacle_detect_face.go),
// which deliberately gates a dual-host dip OUT (qualifying==2 ⇒ Phase-2 corner obstacle, honest-
// reject), this detector admits BOTH host faces independently: S1's two bosses, one on each of the
// fillet's planar faces, are each a genuine, separately-imprintable runout — not a corner pierce.
// A curved (varying-radius) edge sweeps a torus/canal band the straight-cylinder cross-section this
// detector assumes cannot model, so it honest-rejects up front, same as straightFilletEdge's other
// callers.
func detectRunouts(ef edgeFillet, res Resolution) []runoutImprint {
	if ef.varying || !straightFilletEdge(ef, res) {
		return nil
	}
	var out []runoutImprint
	for _, hostIsA := range []bool{true, false} {
		host := ef.b
		if hostIsA {
			host = ef.a
		}
		if im, ok := runoutOnHost(ef, host, hostIsA, res); ok {
			out = append(out, im)
		}
	}
	return out
}

// runoutOnHost checks host qualifies as a runout-imprint candidate: a plane carrying exactly one
// closed-loop hole (the feature footprint). It deliberately carries none of detectObstacleOnHost's
// extra gates (qualifying==1, rebuildableTube, wall-plane checks) — those exist to keep the mid-span
// obstacle REBUILD scoped to a single host; the runout-imprint path never rebuilds a blend surface,
// so a footprint on the other host is simply a second, independent imprint. The actual crossing/dip
// test is the shared bandCrossings sequence, delegated to runoutOnPlane.
func runoutOnHost(ef edgeFillet, host *topo.Face, hostIsA bool, res Resolution) (runoutImprint, bool) {
	pl, ok := host.Geometry().(geom.Plane)
	if !ok {
		return runoutImprint{}, false
	}
	fp, ok := singleHoleEdge(host)
	if !ok || fp.StartVertex() != fp.EndVertex() {
		return runoutImprint{}, false
	}
	return runoutOnPlane(ef, host, hostIsA, pl, fp, res)
}

// runoutOnPlane runs the shared crossing/dip test (bandCrossings, fillet_obstacle_detect_face.go)
// against fp — host's confirmed single closed hole rim — and packages the runout-imprint record on
// success: the footprint rim must cross the receded fillet boundary exactly twice, and the enclosed
// arc must genuinely dip onto the fillet side (not merely bulge toward it).
func runoutOnPlane(ef edgeFillet, host *topo.Face, hostIsA bool, pl geom.Plane, fp *topo.Edge,
	res Resolution) (runoutImprint, bool) {
	_, nodes, flat, back, ok := bandCrossings(ef, hostIsA, pl, fp, res)
	if !ok {
		return runoutImprint{}, false
	}
	return runoutImprint{host, hostIsA, pl, fp, nodes, flat, back}, true
}
