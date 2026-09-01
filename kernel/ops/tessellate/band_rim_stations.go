// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// SHARED-RIM STATION AGREEMENT on a seam-bridged periodic band.
//
// A full cylinder/cone/sphere side — two rim circles bridged by a seam edge used twice — is meshed by
// periodicBandGrid + closedDomainMesh on ONE set of angular stations: the UNION of both rims' own
// boundary samples (bracketPeriod over every boundary point's u). That is watertight only while both
// rims discretize into the SAME stations, and they need not. adaptiveParams bisects until the chord
// sagitta meets the chord tolerance, so a rim of radius r takes 2^ceil(log2(π·√(r/(2·tol)))) stations;
// two rims whose radii straddle a bisection threshold therefore differ by an EXACT factor of 2. The
// union then tiles the COARSER rim at the finer rim's count, while the face on the far side of that
// rim — which correctly took it through discretizeEdge — still tiles it at its own count. Every chord
// of that rim is then unpaired on both sides: a T-junction crack the whole way round.
//
// MEASURED at PropertyQuality (ops.FreeEdgeCount over CalculateBodyFacets(...).Mesh): bfuseblend/A2's
// cone spans rims of r=49.999857 (512 stations) and r=98.081269 (1024); simple/J1's spans r=51.893513
// (512) and r=99.999929 (1024). Each leaked 512+1024 = 1536 free edges — 74 % of the whole corpus's
// mesh leakage — while measuring ZERO at DefaultQuality, whose coarser tolerance puts BOTH rims on 128
// stations so the union is a no-op. That invisibility is why it survived twelve slices; see
// .superpowers/sdd/rim-station-report.md.
//
// The repair keeps the invariant where discretizeEdge's doc states it — on the EDGE. When the grid
// would re-tile a rim, the band is lofted rim-to-rim instead (saddleBandLoftMesh), so each rim row IS
// that rim's shared-edge discretization and the two rows are zipped even at differing counts.
// periodicBandGrid already rejects any band whose non-periodic direction carries more than two values,
// so the gridded band is ALREADY a two-row strip: the loft changes which stations are used, not the
// surface rows, and a band whose rims do agree keeps the grid byte-for-byte.

// unequalRimBandMesh meshes a seam-bridged periodic band whose rims do not all tessellate into the band
// grid's own station count, lofting rim-to-rim so each rim keeps its shared-edge discretization.
// ok=false — keep the grid — when every rim already agrees with the grid (the common case: a cylinder
// wall's two equal-radius rims) or when the face is not a two-rim ruled loft.
//
// Example:
//
//	if us, vs, isBand := periodicBandGrid(s, outer3D, holes3D); isBand {
//		if m, ok := unequalRimBandMesh(f, s, bandGridStations(s, us, vs), q); ok {
//			return m
//		}
//		return closedDomainMesh(s, us, vs)
//	}
func unequalRimBandMesh(f *topo.Face, s geom.Surface, stations int, q Quality) (*Mesh, bool) {
	if !bandGridRetilesARim(f, stations, q) {
		return nil, false
	}
	return saddleBandLoftMesh(f, s, q)
}

// bandGridRetilesARim reports whether gridding the band on `stations` angular stations would tile some
// rim edge at a count other than that rim's OWN discretizeEdge station count — i.e. whether the grid
// breaks the shared-edge invariant on at least one rim. The seam edge bridges the rims and carries no
// station of its own, so it is excluded; a rim needs at least three stations to be a ring at all.
func bandGridRetilesARim(f *topo.Face, stations int, q Quality) bool {
	seam := seamEdgesOf(f)
	for _, e := range f.Edges() {
		if seam[e] {
			continue
		}
		if n := len(dropClosingDup(DiscretizeEdge(e, q))); n >= 3 && n != stations {
			return true
		}
	}
	return false
}

// bandGridStations is the number of DISTINCT stations periodicBandGrid's output grids the band's
// PERIODIC direction at. That direction is the bracketed one (bracketPeriod closes the period by
// repeating the first station as a trailing 2π), so its distinct count is one less than its length.
func bandGridStations(s geom.Surface, us, vs []float64) int {
	if IsPeriodic(s.UDomain()) {
		return len(us) - 1
	}
	return len(vs) - 1
}
