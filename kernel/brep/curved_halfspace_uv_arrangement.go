// SPDX-License-Identifier: GPL-2.0-only

package brep

// General (u,v) arrangement trim of a ruled side (Oblikovati#1405). The analytic walk in
// curved_halfspace_ruled_boundary.go reduces the kept region to a single-valued v-interval
// [lo(u),hi(u)] per azimuth — it reads the surface only through keptV/sectionV, so it cannot trim
// by a MULTI-valued or CLOSED-island imprint (the general curved∩curved case). The arrangement
// replaces that walk with a parameter-space one, and a plane cut becomes the special case where
// the imprint is a single conic.
//
// It is ONE pipeline, in five phases, one file each (split for #2212 — the phases were 1,147 lines
// in this file):
//
//	1. sample  (curved_halfspace_uv_sample.go) project the imprint into (u,v) and sample it into
//	           segments TAGGED with the curve they came from, splitting at the periodic seams;
//	2. band    (curved_halfspace_uv_band.go)   frame the side's own trim, clip the imprint to it,
//	           place the azimuth seam, and decide the material predicate;
//	3. cells   (curved_halfspace_uv_cells.go)  subdivide the band along the imprint through the
//	           shared planar engine (Arrange, arrange2d.go) and classify each cell kept/dropped;
//	4. trace   (curved_halfspace_uv_trace.go)  take the kept region's boundary edges and chain them
//	           into loops, folding the seam so a wrapping region cancels its artificial edges;
//	5. emit    (curved_halfspace_uv_emit.go)   re-emit each traced run as the EXACT arc or conic its
//	           tag names, then group loops into faces by containment and orient them.
//
// The tag carried from phase 1 to phase 5 is what makes the result analytic: without it the
// boundary would come back as the sampled polyline the arrangement worked on.
//
// The ruled side itself — the (u,v) frame these phases work in — is curved_halfspace_uv_side.go.
