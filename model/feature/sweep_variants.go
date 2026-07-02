// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// The sweep definition union (M08 PBI-094, #314): beyond the plain path sweep,
// a profile can ride the path with a guide rail (scaling/orientation), a guide
// surface (orientation locked to a face's normal), piecewise section twists,
// profile orientation modes, and a draft taper; a SOLID sweep drags a tool
// body along the path instead of a profile. The discriminators are the frozen
// api/types sweep enums.

// SweepTwistStation is one row of a pathAndSectionTwists sweep: the twist
// angle (radians) at normalized arclength T ∈ [0, 1]; rows interpolate
// linearly.
type SweepTwistStation struct {
	T     float64
	Angle float64
}

// DefinitionType derives the union discriminator from the populated fields.
func (d *SweepDefinition) DefinitionType() types.SweepDefinitionType {
	switch {
	case d.SolidToolIndex != nil:
		return types.SolidSweepDef
	case d.GuideRail != nil:
		return types.PathAndGuideRailSweepDef
	case len(d.GuideFaceKey) > 0:
		return types.PathAndGuideSurfaceSweepDef
	case len(d.TwistStations) > 0:
		return types.PathAndSectionTwistsSweepDef
	default:
		return types.PathSweepDef
	}
}

// sweepConfig is the resolved per-recompute sweep behavior.
type sweepConfig struct {
	orientation types.SweepProfileOrientation
	alignVec    math.Vector3
	taper       float64
	twistAt     func(t float64) float64
	rail        []math.Point3 // resampled to the path's count; nil ⇒ none
	scaling     types.SweepProfileScaling
	upAt        func(k int) (math.Vector3, bool) // guide-surface up; nil ⇒ none
}

// resolveSweepConfig validates and resolves the union fields against the
// running bodies (the guide face) and the path.
func (s *SweepFeature) resolveSweepConfig(in Input, path []math.Point3) (sweepConfig, error) {
	d := s.def
	cfg := sweepConfig{
		orientation: d.Orientation, alignVec: d.AlignVector,
		taper:   callOrZero(d.Taper),
		twistAt: twistInterpolator(callOrZero(d.Twist), d.TwistStations),
		scaling: d.Scaling,
	}
	if d.GuideRail != nil {
		rail := d.GuideRail()
		if rail == nil || rail.Count() < 2 {
			return cfg, errors.New("sweep: the guide rail needs at least two points")
		}
		cfg.rail = resamplePolyline(rail.Points(), len(path))
	}
	if len(d.GuideFaceKey) > 0 {
		up, err := guideSurfaceUp(in.Bodies, d.GuideFaceKey, path)
		if err != nil {
			return cfg, err
		}
		cfg.upAt = up
	}
	if cfg.orientation == types.AlignToVector && float64(cfg.alignVec.Length()) == 0 {
		return cfg, errors.New("sweep: alignToVector orientation needs a non-zero align vector")
	}
	return cfg, nil
}

// twistInterpolator builds θ(t): the station table when present (piecewise
// linear, clamped), else the simple linear total twist.
func twistInterpolator(total float64, stations []SweepTwistStation) func(float64) float64 {
	if len(stations) == 0 {
		return func(t float64) float64 { return total * t }
	}
	return func(t float64) float64 { return stationAngleAt(stations, t) }
}

// stationAngleAt evaluates the twist station table at t (stations are in
// ascending T order; outside the table the end values hold).
func stationAngleAt(st []SweepTwistStation, t float64) float64 {
	if t <= st[0].T {
		return st[0].Angle
	}
	for i := 0; i+1 < len(st); i++ {
		if t <= st[i+1].T {
			span := st[i+1].T - st[i].T
			if span <= 0 {
				return st[i+1].Angle
			}
			f := (t - st[i].T) / span
			return st[i].Angle + f*(st[i+1].Angle-st[i].Angle)
		}
	}
	return st[len(st)-1].Angle
}

// guideSurfaceUp resolves the guide face and returns the per-station up
// direction: the face's surface normal nearest each path point.
func guideSurfaceUp(bodies []*topo.Body, key []byte, path []math.Point3) (func(int) (math.Vector3, bool), error) {
	face, ok := findFace(bodies, key)
	if !ok {
		return nil, fmt.Errorf("sweep: guide-surface face %q not found on the running bodies", key)
	}
	surf := face.Geometry()
	return func(k int) (math.Vector3, bool) {
		u, v := surf.ParamAt(path[k])
		n := surf.NormalAt(u, v)
		if l := float64(n.Length()); l > 0 {
			return n.Scale(math.Scalar(1 / l)), true
		}
		return math.Vector3{}, false
	}, nil
}

// resamplePolyline re-samples a polyline to n points by arc length.
func resamplePolyline(pts []math.Point3, n int) []math.Point3 {
	cum := make([]float64, len(pts))
	for i := 1; i < len(pts); i++ {
		cum[i] = cum[i-1] + float64(pts[i-1].DistanceTo(pts[i]))
	}
	total := cum[len(cum)-1]
	out := make([]math.Point3, n)
	for i := 0; i < n; i++ {
		out[i] = polylinePointAt(pts, cum, total*float64(i)/float64(n-1))
	}
	return out
}

func polylinePointAt(pts []math.Point3, cum []float64, s float64) math.Point3 {
	for i := 1; i < len(cum); i++ {
		if s <= cum[i] {
			span := cum[i] - cum[i-1]
			if span == 0 {
				return pts[i]
			}
			f := math.Scalar((s - cum[i-1]) / span)
			return pts[i-1].TranslateBy(pts[i-1].VectorTo(pts[i]).Scale(f))
		}
	}
	return pts[len(pts)-1]
}

// sweepSectionsCfg places the profile along the path under the resolved
// config: orientation mode, twist, rail scaling/steering, guide-surface
// steering, and the draft taper.
func sweepSectionsCfg(prof *sketch.Profile, plane sketch.Plane, path []math.Point3, cfg sweepConfig) ([][]math.Point3, error) {
	base := modelPolygon(prof, plane)
	centroid := centroidOf(base)
	normal := plane.Normal()
	tangents := pathTangents(path)
	arc := cumulativeArclength(path)
	steer, err := newSweepSteering(cfg, path, tangents)
	if err != nil {
		return nil, err
	}
	miters := miterDirections(base, centroid, normal)
	sections := make([][]math.Point3, len(path))
	for k := range path {
		sections[k] = sweepSectionAt(base, centroid, normal, miters, path, tangents, arc, cfg, steer, k)
	}
	return sections, nil
}

// sweepSectionAt builds one section: taper offset in profile space (so the
// draft rides the section's rotation), base orientation, steering
// (rail/surface), scaling, twist.
func sweepSectionAt(base []math.Point3, centroid math.Point3, normal math.UnitVector3,
	miters []math.Vector3, path []math.Point3, tangents []math.UnitVector3, arc []float64,
	cfg sweepConfig, steer *sweepSteering, k int) []math.Point3 {
	align := sectionAlign(cfg, normal, tangents, k)
	axis := sectionAxis(cfg, normal, tangents, k)
	t := arc[k] / arc[len(arc)-1]
	twist := cfg.twistAt(t)
	draft := stdmath.Tan(cfg.taper) * arc[k]
	sec := make([]math.Point3, len(base))
	for i, p := range base {
		rel := centroid.VectorTo(p)
		if draft != 0 {
			rel = rel.Add(miters[i].Scale(math.Scalar(draft)))
		}
		v := align.TransformVector(rel)
		v = steer.apply(v, k, axis)
		if twist != 0 {
			v = math.Rotation4(twist, axis, math.P3(0, 0, 0)).TransformVector(v)
		}
		sec[i] = path[k].TranslateBy(v)
	}
	return sec
}

// miterDirections precomputes, per profile vertex, the outward offset
// direction scaled so a unit move offsets every WALL by exactly one unit —
// the constant-draft taper semantics (a plain radial offset would move a
// square's corners √2 farther than its walls, overstating the draft on flats
// and understating it at corners).
func miterDirections(base []math.Point3, centroid math.Point3, normal math.UnitVector3) []math.Vector3 {
	n := len(base)
	out := make([]math.Vector3, n)
	for i := range base {
		prev, next := base[(i-1+n)%n], base[(i+1)%n]
		n1 := edgeOutNormal(prev, base[i], centroid, normal)
		n2 := edgeOutNormal(base[i], next, centroid, normal)
		bis := n1.Add(n2)
		l := float64(bis.Length())
		if l == 0 { // straight-through vertex: either normal works
			out[i] = n1
			continue
		}
		bis = bis.Scale(math.Scalar(1 / l))
		cosHalf := float64(bis.Dot(n1))
		if cosHalf < 0.1 {
			cosHalf = 0.1 // cap the miter at very sharp corners (≈84°+ half-angle)
		}
		out[i] = bis.Scale(math.Scalar(1 / cosHalf))
	}
	return out
}

// edgeOutNormal is the in-plane unit normal of edge a→b pointing away from
// the centroid.
func edgeOutNormal(a, b, centroid math.Point3, normal math.UnitVector3) math.Vector3 {
	e := a.VectorTo(b)
	nrm := e.Cross(normal.AsVector())
	if l := float64(nrm.Length()); l > 0 {
		nrm = nrm.Scale(math.Scalar(1 / l))
	}
	mid := a.TranslateBy(e.Scale(0.5))
	if float64(centroid.VectorTo(mid).Dot(nrm)) < 0 {
		nrm = nrm.Scale(-1)
	}
	return nrm
}

// sectionAlign is the base profile rotation for the orientation mode.
func sectionAlign(cfg sweepConfig, normal math.UnitVector3, tangents []math.UnitVector3, k int) math.Matrix4 {
	switch cfg.orientation {
	case types.ParallelToOriginalProfile:
		return math.Identity4()
	case types.AlignToVector:
		u, err := math.UnitVector3FromVector(cfg.alignVec)
		if err != nil {
			return math.Identity4()
		}
		return math.RotateBetween(normal, u)
	default: // NormalToPath
		return math.RotateBetween(normal, tangents[k])
	}
}

// sectionAxis is the twist/steering axis: the path tangent for normal-to-path
// sweeps, the (fixed) section normal otherwise.
func sectionAxis(cfg sweepConfig, normal math.UnitVector3, tangents []math.UnitVector3, k int) math.UnitVector3 {
	switch cfg.orientation {
	case types.ParallelToOriginalProfile:
		return normal
	case types.AlignToVector:
		if u, err := math.UnitVector3FromVector(cfg.alignVec); err == nil {
			return u
		}
		return normal
	default:
		return tangents[k]
	}
}

// cumulativeArclength returns the running arclength at each path point.
func cumulativeArclength(path []math.Point3) []float64 {
	out := make([]float64, len(path))
	for i := 1; i < len(path); i++ {
		out[i] = out[i-1] + float64(path[i-1].DistanceTo(path[i]))
	}
	if out[len(out)-1] == 0 {
		out[len(out)-1] = 1 // degenerate single-point guard for the t division
	}
	return out
}

// sweepSteering tracks a reference direction (rail vector or surface normal)
// about the section axis: each section rotates so its local reference axis
// follows the guide, and rail sweeps scale with the rail distance.
type sweepSteering struct {
	refs   []math.Vector3 // per-station target direction ⊥ axis (zero ⇒ none)
	dists  []float64      // rail distance per station (scaling)
	scale  types.SweepProfileScaling
	hasRef bool
}

// newSweepSteering precomputes the per-station steering targets.
func newSweepSteering(cfg sweepConfig, path []math.Point3, tangents []math.UnitVector3) (*sweepSteering, error) {
	st := &sweepSteering{scale: cfg.scaling}
	switch {
	case cfg.rail != nil:
		if err := st.fromRail(cfg, path, tangents); err != nil {
			return nil, err
		}
	case cfg.upAt != nil:
		st.fromSurface(cfg, path, tangents)
	}
	return st, nil
}

func (st *sweepSteering) fromRail(cfg sweepConfig, path []math.Point3, tangents []math.UnitVector3) error {
	st.refs = make([]math.Vector3, len(path))
	st.dists = make([]float64, len(path))
	for k := range path {
		ref := perpComponent(path[k].VectorTo(cfg.rail[k]), tangents[k])
		d := float64(ref.Length())
		if d < 1e-12 {
			return fmt.Errorf("sweep: guide rail touches the path at station %d — rail and path must stay apart", k)
		}
		st.refs[k], st.dists[k] = ref, d
	}
	st.hasRef = true
	return nil
}

func (st *sweepSteering) fromSurface(cfg sweepConfig, path []math.Point3, tangents []math.UnitVector3) {
	st.refs = make([]math.Vector3, len(path))
	for k := range path {
		if n, ok := cfg.upAt(k); ok {
			st.refs[k] = perpComponent(n, tangents[k])
		}
	}
	st.scale = types.NoProfileScaling
	st.hasRef = true
}

// apply steers one displacement vector at station k: rotate the section so the
// station-0 reference direction tracks the station-k one, then scale per the
// rail mode.
func (st *sweepSteering) apply(v math.Vector3, k int, axis math.UnitVector3) math.Vector3 {
	if !st.hasRef || float64(st.refs[0].Length()) == 0 || float64(st.refs[k].Length()) == 0 {
		return v
	}
	phi := signedAngleAbout(st.refs[0], st.refs[k], axis)
	if phi != 0 {
		v = math.Rotation4(phi, axis, math.P3(0, 0, 0)).TransformVector(v)
	}
	return st.scaleBy(v, k)
}

// scaleBy applies the rail scaling: XY scales the whole section with the rail
// distance, X only the component along the rail direction, none leaves size.
func (st *sweepSteering) scaleBy(v math.Vector3, k int) math.Vector3 {
	if st.dists == nil || st.scale == types.NoProfileScaling {
		return v
	}
	f := st.dists[k] / st.dists[0]
	if st.scale == types.XProfileScaling {
		dir := st.refs[k].Scale(math.Scalar(1 / st.refs[k].Length()))
		along := dir.Scale(v.Dot(dir))
		return v.Sub(along).Add(along.Scale(math.Scalar(f)))
	}
	return v.Scale(math.Scalar(f)) // XY (the default)
}

// perpComponent removes the axis-parallel part of v.
func perpComponent(v math.Vector3, axis math.UnitVector3) math.Vector3 {
	a := axis.AsVector()
	return v.Sub(a.Scale(v.Dot(a)))
}

// signedAngleAbout is the signed rotation from a to b about axis (both ⊥ axis).
func signedAngleAbout(a, b math.Vector3, axis math.UnitVector3) float64 {
	la, lb := float64(a.Length()), float64(b.Length())
	if la == 0 || lb == 0 {
		return 0
	}
	cos := float64(a.Dot(b)) / (la * lb)
	cos = stdmath.Max(-1, stdmath.Min(1, cos))
	angle := stdmath.Acos(cos)
	if float64(a.Cross(b).Dot(axis.AsVector())) < 0 {
		angle = -angle
	}
	return angle
}

// --- solid sweep -------------------------------------------------------------

// recomputeSolidSweep drags the tool BODY along the path: the swept volume is
// the union of tool stamps at path samples dense enough that consecutive
// stamps overlap (step ≤ half the tool's smallest extent), translation only —
// the discrete swept-envelope construction.
func (s *SweepFeature) recomputeSolidSweep(in Input) (Output, error) {
	idx := *s.def.SolidToolIndex
	if idx < 0 || idx >= len(in.Bodies) {
		return Output{}, fmt.Errorf("sweep: solid tool body %d out of range (%d bodies)", idx, len(in.Bodies))
	}
	toolSrc := in.Bodies[idx]
	path, err := s.resolvePathPoints()
	if err != nil {
		return Output{}, err
	}
	swept, err := sweptToolEnvelope(toolSrc, path, featOr(s.featName, "sweep"), in.Diag)
	if err != nil {
		return Output{}, err
	}
	s.tool = swept
	bodies, err := combine(in, swept, s.def.Operation)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// resolvePathPoints resolves and validates the definition's path.
func (s *SweepFeature) resolvePathPoints() ([]math.Point3, error) {
	if s.def.Path == nil {
		return nil, errors.New("sweep: path needs at least two points")
	}
	path := s.def.Path()
	if path == nil || path.Count() < 2 {
		return nil, errors.New("sweep: path needs at least two points")
	}
	return path.Points(), nil
}

// sweptToolEnvelope unions translated tool stamps along the path. rec collects the stamp
// unions' boolean-fallback diagnostics (#1601; nil discards).
func sweptToolEnvelope(tool *topo.Body, path []math.Point3, feat string, rec *diag.Recorder) (*topo.Body, error) {
	samples := stampStations(tool, path)
	origin := samples[0]
	swept, err := stampAt(tool, origin, origin, feat, 0)
	if err != nil {
		return nil, err
	}
	for i := 1; i < len(samples); i++ {
		stamp, err := stampAt(tool, origin, samples[i], feat, i)
		if err != nil {
			return nil, err
		}
		if swept, err = ops.BooleanWithDiagnostics(ops.Join, swept, stamp, rec); err != nil {
			return nil, fmt.Errorf("sweep: stamp %d union failed: %w", i, err)
		}
	}
	return swept, nil
}

// stampStations resamples the path so consecutive stamps overlap: the step is
// half the tool's smallest range-box extent.
func stampStations(tool *topo.Body, path []math.Point3) []math.Point3 {
	d := tool.RangeBox().Diagonal()
	minExtent := stdmath.Min(stdmath.Min(stdmath.Abs(float64(d.X)), stdmath.Abs(float64(d.Y))), stdmath.Abs(float64(d.Z)))
	cum := cumulativeArclength(path)
	total := cum[len(cum)-1]
	step := minExtent / 2
	if step <= 0 {
		step = total / 8
	}
	n := int(stdmath.Ceil(total/step)) + 1
	if n < len(path) {
		n = len(path)
	}
	return resamplePolyline(path, n)
}

// stampAt translates the tool from the path origin to a station (identity
// lineage at station 0 so the first stamp IS the tool's footprint there).
func stampAt(tool *topo.Body, origin, at math.Point3, feat string, i int) (*topo.Body, error) {
	m := math.Translation4(origin.VectorTo(at))
	return ops.TransformBody(tool, m, func(l topo.Lineage) topo.Lineage {
		return topo.NewLineage(append(l.Tokens(), topo.Tok(feat, "stamp", i))...)
	})
}
