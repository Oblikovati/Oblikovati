// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"strings"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
)

// The CURVED half of the boolean: the general per-face pipeline and the guards that decide whether to
// trust its result (split out of boolean.go for #2215).
//
// There is ONE path here. Until ADR-0061 stage 4 this file opened with curvedExactPaths, an ordered
// first-fit list of 26 bespoke recognizers tried BEFORE the general pipeline; the ground rules forbid
// that shape — "dispatch is a classification that selects exactly one path" — and every pair those
// recognizers claimed is now built by brep's per-face dispatch. What they were (the ruled crossings, the
// equal-radius Steinmetz family, the drill through-hole and the cylinder boss, the cap crossings, the
// coaxial ball and rod) and why each is no longer a separate handler is recorded in ADR-0045 and
// ADR-0061; the brep drivers behind them are deleted with them.
//
// The guards below are why a wrong result does not ship: the body must be a valid closed solid, its
// faces must pass the Requicha membership certificate, no face may be wound against its outward normal,
// and the volume must land inside the Requicha bracket. A pair the pipeline cannot model is refused BY
// NAME (ErrUnmodelledBoolean); nothing is faceted behind the caller's back.

// curvedExactBoolean runs brep's per-face-dispatch boolean (ADR-0058) on a pair carrying at least one
// curved face and adopts the result only as a valid, correctly-wound solid. It is the ONLY curved path:
// what brep's scope gate declines (ErrUnsupportedMixedBoolean) the caller refuses by name, where it once
// fell first to the 26 recognizers and then to the mesh reconstruction (ADR-0061 stages 4 and 6).
//
// An all-planar pair declines here (ok=false) and keeps its own guarded planar pipeline downstream,
// byte-for-byte.
func curvedExactBoolean(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	if !hasCurvedFace(target) && !hasCurvedFace(tool) {
		return nil, false
	}
	bop, ok := toBrepOp(op)
	if !ok {
		return nil, false
	}
	body, err := brep.BooleanDiag(bop, target, tool, rec)
	if err != nil || body == nil || !Validate(body).ValidSolid() {
		return nil, false
	}
	if inverted, found := invertedFace(body); found {
		rec.Recordf(CodeBooleanWindingReject, diag.Defect,
			"curved %s general result has a face wound against its outward normal (%q): declining it", op, inverted.ReferenceKey())
		return nil, false
	}
	return body, true
}

// hasCurvedFace reports whether a body carries a non-planar (analytic curved) face.
func hasCurvedFace(b *topo.Body) bool {
	for _, f := range b.Faces() {
		if _, planar := f.Geometry().(geom.Plane); !planar {
			return true
		}
	}
	return false
}

// invertedFace returns a face of the body whose loops wind against its outward normal — the emission
// post-condition the per-edge validity test cannot see (brep.FaceWindingConsistent). A face the
// certificate cannot read is not reported: the gate refuses only what it can prove.
func invertedFace(b *topo.Body) (*topo.Face, bool) {
	for _, f := range b.Faces() {
		if ok, certain := brep.FaceWindingConsistent(f); certain && !ok {
			return f, true
		}
	}
	return nil, false
}

// CodeBooleanWindingReject marks a curved boolean result refused because one of its faces is wound
// against its outward normal. Validate's per-edge test admits such a body — two faces inverted
// together across the edge they share stay pairwise consistent — and the tessellator then meshes the
// face's complement while the analytic integrator, which signs a loop from its own boundary integral,
// still reports the right volume. The torus tangent cut shipped so for every axis (ADR-0061 stage 4).
const CodeBooleanWindingReject diag.Code = "boolean.winding-reject"

// CodeBooleanAnalyticVolumeReject marks a curved analytic boolean whose result fell OUTSIDE the
// Requicha two-sided volume bracket (#1601): the recognizer produced a valid body of materially
// wrong volume (kept the wrong lobe, removed too much), so it is rejected and the operation falls
// back to the guarded planar/CSG path instead of shipping the wrong analytic solid. A tracked
// defect — before this, the analytic short-circuit bypassed the volume guard the planar path gets.
const CodeBooleanAnalyticVolumeReject diag.Code = "boolean.analytic-volume-reject"

// curvedVolumeGuardFraction is the volume bracket's tolerance as a fraction of the larger operand's
// volume, for the case where a body's volume the analytic integrator DECLINES and the tessellation
// therefore measures: that number under-measures a curved body by ~0.6% (a cylinder) up to a bounded
// few percent, and the bracket has to absorb the deficit. It no longer applies when the volumes are
// exact (M48/C3 #3445: a 10% window demoted correct analytic results to facets because the MESHER,
// not the boolean, was wrong).
const curvedVolumeGuardFraction = 0.10 // tol:calibrated — bracket margin for a tessellated fallback volume

// curvedGuardBracketOverride, when non-nil, REPLACES the computed volume-bracket tolerance.
// Production leaves it nil; a test sets it to a large negative value so even a correct result reads
// as out of bracket, which drives the reject-and-diagnose path end to end without needing a
// recognizer bug to reproduce. It replaces rather than scales because the analytic bracket's own
// tolerance is a resolution cube — scaling that by −1 leaves a number far too small to force a
// violation, so the hook silently stopped working when the bracket became exact.
var curvedGuardBracketOverride *float64

// curvedExactGuarded returns an exact analytic curved boolean only when a path applies AND the
// result is CERTIFIED: every face passes Requicha's membership rule against the operands
// (certifyBooleanFaces), and the volume lands inside the Requicha two-sided bracket. The analytic
// short-circuit otherwise bypasses the guard the planar path gets, so a recognizer that
// mis-classifies to a valid body of materially wrong shape would ship silently. The per-face gate is
// the proof and the volume bracket the smoke test — a body can hold the right amount of material in
// the wrong place. When either rejects, this records a Defect and declines (ok=false) so
// booleanGeneral falls through to the guarded planar/CSG path. On acceptance it restores
// original-edge identity (ADR-0043) like the planar path.
//
// sizes is the pair's SIZE classification, already decided by whichever public entry the call came
// through. This core does not re-run it — it used to, which made the predicate run twice for every
// curved pair reached through booleanGeneralExact (#3524) — and passes it on to the certificate,
// which needs the same pair resolution.
func curvedExactGuarded(op PartFeatureOperation, target, tool *topo.Body, sizes operandSizes, rec *diag.Recorder) (*topo.Body, bool) {
	body, ok := curvedExactBoolean(op, target, tool, rec)
	if !ok {
		declineCurvedExact(op, target, tool, rec)
		return nil, false
	}
	// Validate stays HERE, at the exit that returns the body, rather than inside the gate below: the
	// post-condition of a public operation has to be visible on the path its result travels
	// (archguard TestExportedOpsValidateTheirResult follows only the calls whose result is returned).
	if !Validate(body).ValidSolid() {
		rec.Recordf(CodeBooleanAnalyticInvalid, diag.Defect,
			"curved %s analytic result is not a valid closed solid: falling back to the guarded path", op)
		return nil, false
	}
	if curvedResultRejected(op, target, tool, body, sizes, rec) {
		return nil, false
	}
	body.InheritOriginalEdges(append(append([]*topo.Edge(nil), target.Edges()...), tool.Edges()...))
	return body, true
}

// curvedResultRejected is the acceptance gate curvedExactGuarded applies to a VALID built result, in
// the order the ground rules put them: the per-face membership certificate is the PROOF, winding is a
// post-condition the certificate cannot see, and the whole-body volume bracket is the closing smoke
// test. Each rejection records its own Defect naming which certificate refused, so a demotion says WHY
// rather than only that it happened, and an ACCEPTANCE the certificate could not take in full records
// how much of the body it read. Validity is checked by the caller, at the exit (see there).
func curvedResultRejected(op PartFeatureOperation, target, tool, body *topo.Body, sizes operandSizes, rec *diag.Recorder) bool {
	if faceCertificateRejected(op, target, tool, body, sizes.res, rec) {
		return true
	}
	if inverted, found := invertedFace(body); found {
		rec.Recordf(CodeBooleanWindingReject, diag.Defect,
			"curved %s analytic result has a face wound against its outward normal (%q): falling back to the guarded path", op, inverted.ReferenceKey())
		return true
	}
	vols, m := boolVolumes(target, tool, body)
	recordMovedVolumeOutOfToolBracket(op, vols, m, rec)
	return curvedVolumeRejected(op, target, tool, vols, m.operandsAnalytic(), rec)
}

// faceCertificateRejected is the per-face half of the acceptance gate: the evidence certifyBooleanFaces
// collects, judged. Each refusal records its OWN Defect, so a demotion says which of the three
// quantities disagreed — where a face lies, or how much boundary the result shows — rather than only
// that something did.
func faceCertificateRejected(op PartFeatureOperation, target, tool, body *topo.Body, res Resolution, rec *diag.Recorder) bool {
	ev := certifyBooleanFaces(op, target, tool, body, res)
	// FIRST, and unconditionally: how much of the body the certificate could read is most worth
	// knowing on the paths that go on to refuse it, and a `||` short-circuit hid it from exactly those.
	recordUnexaminedFaces(op, body, ev, rec)
	if !ev.kept {
		rec.Recordf(CodeBooleanAnalyticFaceReject, diag.Defect,
			"curved %s analytic result has a face the operands do not account for: falling back to the guarded path", op)
		return true
	}
	return overclaimRejected(op, target, tool, ev, rec)
}

// overclaimRejected refuses a result showing more boundary than its operands have between them.
func overclaimRejected(op PartFeatureOperation, target, tool *topo.Body, ev faceEvidence, rec *diag.Recorder) bool {
	available, over := ev.overclaimsItsOperands(target, tool)
	if !over {
		return false
	}
	rec.Recordf(CodeBooleanAnalyticFaceReject, diag.Defect,
		"curved %s analytic result shows %g of boundary where its operands have %g between them: a boolean "+
			"trims its operands and cannot grow them — falling back to the guarded path", op, ev.claimed, available)
	return true
}

// recordUnexaminedFaces reports how much of the body the certificate could not read: the faces with no
// interior point, which the membership rule was never applied to, and the faces with no analytic area,
// which the boundary bound does not cover. It never refuses — the gate disproves, it does not demand a
// probe.
func recordUnexaminedFaces(op PartFeatureOperation, body *topo.Body, ev faceEvidence, rec *diag.Recorder) {
	if ev.unprobed == 0 && ev.unmeasured == 0 {
		return
	}
	rec.Recordf(CodeBooleanFaceNotProbed, diag.Warning,
		"curved %s analytic result: of %d faces, %d had no interior point (the membership rule was not "+
			"applied to them) and %d had no analytic area (the boundary bound does not cover them)",
		op, len(body.Faces()), ev.unprobed, ev.unmeasured)
}

// recordMovedVolumeOutOfToolBracket reports the material the operation MOVED against the tool that
// moved it — the per-operation half of Requicha's rule taken at the TOOL's scale instead of the
// model's, which is the only scale at which a small feature is visible at all.
//
// The bracket above it is model-relative: on the RING pair its tolerance is
// ResolutionForBodies(ring, drill).Volume() = 6.464e-3 mm³, which is 686x the material a 1e-3 bore
// removes and 2740x a 6.31e-4 bore's. It is therefore structurally blind to every feature under
// ~6.5e-3 mm³ on a 20 mm part, and it rejected the G8 band only because the ARTEFACT it was reading
// (2.839, the ring's tessellation deficit) happened to be 440x its tolerance. With that artefact
// gone (#3516) the band ships, and at bore 8.913e-4 it shipped a Cut whose measured removal is
// NEGATIVE — the result measuring larger than the target it was cut from — with err=nil and an empty
// recorder.
//
// This is not an accuracy statement and needs no tolerance of its own: no operation can move more
// material than its tool holds, and none can move a negative amount, so a violation is a
// CONTRADICTION and saying so cannot over-claim. It is recorded, not refused: the body is the exact
// section (#3516 measured its faces and edges), and what is wrong is the number, which the caller
// now sees instead of storing silently.
func recordMovedVolumeOutOfToolBracket(op PartFeatureOperation, vols volumeTriple, src volumeSource, rec *diag.Recorder) {
	tv, wv, bv := vols.target, vols.tool, vols.body
	moved, ok := movedVolume(op, tv, bv)
	if !ok {
		return
	}
	if !src.allAnalytic() {
		rec.Recordf(CodeBooleanVolumeNotBracketed, diag.Warning,
			"curved %s: %s measured by tessellation, so the result's moved volume was NOT bracketed against "+
				"its tool — a mesh deficit is orders above this bracket's slack and would read as a contradiction",
			op, src.meshed())
		return
	}
	slack := movedVolumeSlack * wv
	if moved >= -slack && moved <= wv+slack {
		return
	}
	rec.Recordf(CodeBooleanMovedVolumeOutOfToolBracket, diag.Defect,
		"%s moved %g of material with a tool holding %g (V(A)=%g V(result)=%g): outside [0, V(tool)], "+
			"which no %s can be — the result's measured volume is wrong, not merely imprecise",
		op, moved, wv, tv, bv, op)
}

// movedVolumeSlack is how far outside [0, V(tool)] a measured move may sit before it is a
// contradiction rather than a rounding, as a fraction of the TOOL's own volume. It is not a fitted
// threshold: swept across the whole of kernel/ops at slack 0, the excursions fall into two
// populations with nothing between them. NINE are correct results that consume their tool WHOLE and
// overshoot at the last ulp, the largest by 5.6e-15 of V(tool) — a cut moving 12.566370614359244
// against a tool holding 12.566370614359172. Two are the RING band's real violations, 1.94e-2 BELOW
// zero and 3.21e-1 above one. Any slack in (5.6e-15, 1.94e-2) silences the nine and keeps the two;
// that window is 12.5 orders wide, and 1e-9 sits within a factor of 3 of its geometric centre. The
// normalisation is what makes it scale-free: the excursion is measured against V(tool), so the
// benign population stays at ulp scale whatever the model's size.
const movedVolumeSlack = 1e-9 // tol:calibrated — plateau (5.6e-15, 1.94e-2), measured across kernel/ops

// movedVolume is how much material the operation moved, in the direction its own definition moves it:
// what a Cut REMOVED from the target, what a Join ADDED to it, and what an Intersect KEPT. Requicha
// bounds all three the same way — by the tool — so one bracket serves them and there is no per-op
// arm to keep in step. ok is false for an operation with no membership rule.
func movedVolume(op PartFeatureOperation, targetVol, bodyVol float64) (float64, bool) {
	switch op {
	case Cut:
		return targetVol - bodyVol, true
	case Join:
		return bodyVol - targetVol, true
	case Intersect:
		return bodyVol, true
	}
	return 0, false
}

// CodeBooleanMovedVolumeOutOfToolBracket marks a boolean whose result moved material the tool cannot
// account for: a negative removal, or more material than the tool holds. Unlike the model-relative
// bracket beside it, this one is at the TOOL's scale, so it still sees a feature far below the
// model's resolution cube — which is exactly where a wrong measurement used to ship in silence
// (Oblikovati/Oblikovati#3516).
const CodeBooleanMovedVolumeOutOfToolBracket diag.Code = "boolean.moved-volume-out-of-tool-bracket"

// curvedVolumeRejected is the acceptance gate's last stage: the Requicha two-sided volume bracket,
// split out so each stage stays one decision. It takes the volumes its caller already measured, so
// the two brackets read ONE measurement of each body rather than two of each.
func curvedVolumeRejected(op PartFeatureOperation, target, tool *topo.Body, vols volumeTriple, exact bool, rec *diag.Recorder) bool {
	if !volumeOutOfBracket(op, vols, curvedGuardTolerance(target, tool, vols, exact)) {
		return false
	}
	rec.Recordf(CodeBooleanAnalyticVolumeReject, diag.Defect,
		"curved %s analytic result volume %g outside the Requicha bracket (V(A)=%g V(B)=%g): falling back to the guarded path",
		op, vols.body, vols.target, vols.tool)
	return true
}

// declineCurvedExact records the NAMED decline when no exact analytic path claims a configuration that
// carries curved geometry. Until now this was the one exit of the guarded entry that said nothing: the
// three REJECTIONS below each record, while "no path applied" returned silently and the caller quietly
// produced triangle soup. The ground rule is that a fallback is a diag.Defect that reaches feature
// health, the API and the UI — a demotion the user cannot see is the failure mode ADR-0061 stage 6 is
// named for, and the torus figure-eight's three faceted rows reached the corpus through exactly this
// silence.
//
// Silence stays correct for an all-planar pair: the planar B-rep path takes those exactly, so declining
// the curved paths costs nothing and saying so on every boolean in the system would be noise.
func declineCurvedExact(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) {
	if !hasCurvedFace(target) && !hasCurvedFace(tool) {
		return
	}
	rec.Recordf(CodeBooleanNoExactCurvedPath, diag.Defect,
		"curved %s: no exact analytic path claims this configuration (target %d faces, tool %d faces); the result will be faceted",
		op, len(target.Faces()), len(tool.Faces()))
}

// CodeBooleanNoExactCurvedPath marks a boolean with a curved operand that no exact analytic path
// claimed, so the result comes from the faceted fallback. A tracked degradation, not an error: the
// operation succeeds and the body is valid, but it is a tessellation of the answer rather than the
// answer.
const CodeBooleanNoExactCurvedPath diag.Code = "boolean.no-exact-curved-path"

// CodeBooleanAnalyticInvalid marks a curved analytic boolean whose result is not a valid closed solid.
// Validate is the post-condition of every public kernel operation, and this entry had none: the inner
// paths that DO validate (mixedPassThroughBoolean) covered most of the surface, so a recognizer that
// returned a torn body shipped it to whichever caller did not re-check — which, once the public
// CurvedBoolean entries took this guarded path, is none of them (ADR-0061). A tracked degradation: the
// analytic result is refused and the operation falls to the guarded planar path.
const CodeBooleanAnalyticInvalid diag.Code = "boolean.analytic-invalid"

// CodeBooleanFaceNotProbed marks a curved analytic result the per-face certificate could not examine
// in full: a face of it has no interior point to classify, so the membership rule was never applied
// there. It is a Warning, not a Defect — every face that COULD be read passed, and the result is
// probably fine — but it is the honest size of the proof: "certified" on a body with unprobed faces
// means fewer faces were checked than the body has (Oblikovati/Oblikovati#3516).
const CodeBooleanFaceNotProbed diag.Code = "boolean.face-not-probed"

// CodeBooleanAnalyticFaceReject marks a curved analytic boolean whose result carried a face the
// operands cannot account for under the operation's membership rule — a face on neither operand's
// boundary, or on the side the operation removes. Unlike a volume miss this localizes the fault to
// a face, and it catches a wrong lobe that happens to have the right volume.
const CodeBooleanAnalyticFaceReject diag.Code = "boolean.analytic-face-reject"

// curvedGuardTolerance is the volume bracket's slack: the model-relative resolution cube when both
// operands integrate analytically (the volumes are exact, so nothing wider is justified), widened to
// curvedVolumeGuardFraction of the larger operand when either fell back to the tessellation, whose
// chord deficit the bracket must then absorb.
func curvedGuardTolerance(target, tool *topo.Body, vols volumeTriple, exact bool) float64 {
	if curvedGuardBracketOverride != nil {
		return *curvedGuardBracketOverride
	}
	if exact {
		return ResolutionForBodies(target, tool).Volume()
	}
	return curvedVolumeGuardFraction * max(vols.target, vols.tool)
}

// volumeSource records, for each of the three bodies the acceptance brackets measure, whether its
// volume came from the analytic B-rep or from a tessellation. It replaces a separate
// analyticVolumesExact pass that re-integrated both operands to ask the same question the
// measurement had already answered.
type volumeSource struct{ target, tool, body bool }

// operandsAnalytic reports whether both OPERANDS integrated analytically, so their volumes carry no
// tessellation deficit for the Requicha bracket to absorb.
func (m volumeSource) operandsAnalytic() bool { return m.target && m.tool }

// allAnalytic reports whether all three did. The tool-scale bracket needs the stronger form: it has
// no tolerance to absorb a deficit with, so comparing a meshed number with an analytic one — the
// artefact #3516 exists to correct — would read a ~1e-2 mesh deficit as a contradiction against a
// 1e-9 slack.
//
// How much it is doing today, measured over kernel/ops/boolean rather than asserted: **15 of 437
// calls** reach the bracket with at least one meshed volume (12 cuts, 2 intersects, 1 join) and are
// skipped. Removing the guard takes the package's firings from 8 to 9 — and the ONE extra is the
// synthetic row planted to prove the guard, not a body: with that row skipped as well, guard off
// fires exactly 3 times, the two RING violations and the deliberate stub. **So no corpus body is
// mis-Defected today.** The guard prevents a class this corpus does not currently exhibit, and the
// 15 skips it produces are reported rather than silent (CodeBooleanVolumeNotBracketed) — a gate that
// does not run is a proof nobody can size, which is the shape #3516 was filed for.
func (m volumeSource) allAnalytic() bool { return m.operandsAnalytic() && m.body }

// meshed names the bodies whose volume came from a tessellation, for the diagnostic that says the
// bracket did not run. It never returns "" where allAnalytic is false.
func (m volumeSource) meshed() string {
	var out []string
	for i, analytic := range [3]bool{m.target, m.tool, m.body} {
		if !analytic {
			out = append(out, [3]string{"the target", "the tool", "the result"}[i])
		}
	}
	return strings.Join(out, " and ")
}

// CodeBooleanVolumeNotBracketed marks a curved analytic result whose moved volume was NOT checked
// against its tool, because one of the three volumes came from a tessellation rather than from the
// analytic B-rep. It is a Warning, not a Defect: nothing is known to be wrong, and that is the point
// — the acceptance gate did not run, and a gate that silently does not run is exactly the blind spot
// this issue exists to close (Oblikovati/Oblikovati#3516).
const CodeBooleanVolumeNotBracketed diag.Code = "boolean.volume-not-bracketed"

// CurvedBoolean attempts the exact analytic curved boolean and reports whether it applied. It is SAFE to
// call on any operands — each path declines (ok=false) when it does not handle (op, target, tool), and none
// hangs (unlike the planar B-rep boolean, which loops on a full periodic curved face). The model layer uses
// it to combine a still-analytic primitive (a revolved torus, an extruded cylinder) by its curved faces
// before falling back to faceting the operands for the planar path (#129).
//
// It is the GUARDED entry (curvedExactGuarded), the same one booleanGeneralExact takes: a result must pass
// the per-face membership certificate and the Requicha volume bracket, or this declines. The public entry
// used to call curvedExactBoolean directly, so a recognizer that over-matched to a valid body of materially
// wrong shape shipped to the feature layer uncertified while the identical call inside the kernel was
// certified. Only the feature layer's face-COUNT gate stood between a wrong result and the model — which is
// why widening that gate to a classification (ADR-0061) had to close this seam first: one operation, one
// certification, whoever calls it.
func CurvedBoolean(op PartFeatureOperation, target, tool *topo.Body) (*topo.Body, bool) {
	return CurvedBooleanWithDiagnostics(op, target, tool, nil)
}

// CurvedBooleanWithDiagnostics is [CurvedBoolean] with a diagnostic recorder (nil to discard):
// the exact paths record imprint-quality diagnostics (#1404) and their internal fallbacks, so a
// feature-level caller carries the kernel's quality signal instead of dropping it (#1601).
func CurvedBooleanWithDiagnostics(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	// The size classification runs at THIS entry as well as at BooleanWithDiagnostics, because this is
	// a PUBLIC entry too and a sub-resolution pair reaching the core is certified as "nothing removed"
	// — a valid body, every face accounted for, and a Cut volume the Requicha bracket admits (ADR-0061
	// stage 6). It runs at the entry and NOT in the core, so one call classifies the pair exactly once
	// however it arrived (#3524).
	sizes, err := classifyOperandSize(op, target, tool, rec)
	if err != nil {
		return nil, false
	}
	return curvedExactGuarded(op, target, tool, sizes, rec)
}

// shouldFallbackBoolean decides whether a result must be abandoned for the next path. Validity comes
// first (an invalid body is never a result), then the per-face membership certificate — the proof
// that the faces are the ones this operation keeps — and only then the whole-body volume bracket,
// which is a cheap smoke test for what the per-face gate could not probe (M48/C3 #3446/#3447).
func shouldFallbackBoolean(op PartFeatureOperation, target, tool, body *topo.Body, sizes operandSizes) bool {
	if !Validate(body).ValidSolid() {
		return true
	}
	if !certifyBooleanFaces(op, target, tool, body, sizes.res).kept {
		return true
	}
	return invalidBooleanVolume(op, target, tool, body)
}

func invalidBooleanVolume(op PartFeatureOperation, target, tool, body *topo.Body) bool {
	// Model-relative volume tolerance (ADR-0042): scales with the operands' size³ so
	// the result-volume sanity check is faithful at any scale, not just ~cm parts. The
	// planar path's arithmetic is exact-plane, so a tight resolution-cube tol is right.
	vols, _ := boolVolumes(target, tool, body)
	return volumeOutOfBracket(op, vols, ResolutionForBodies(target, tool).Volume())
}

// boolVolumes measures the target, tool and result volumes for the acceptance brackets, and reports
// for each whether the number is analytic or meshed. All three integrate the ANALYTIC B-rep where
// they can (M48/C3 #3448), so the bracket no longer compares three tessellations whose chord deficits
// it had to be widened to absorb. The quality is shared and only reaches a body the analytic path
// declines, where having all three measured the same way still lets their deficits partly cancel.
//
// The SOURCE comes back with the numbers because a bracket that does not know which it holds can
// compare a meshed volume with an analytic one, which is precisely the artefact #3516 was filed for.
// Reading it here costs nothing: it is the choice BodyGeometryProperties already makes internally.
func boolVolumes(target, tool, body *topo.Body) (vols volumeTriple, src volumeSource) {
	q := DefaultQuality()
	vols.target, src.target = analyticOrMeshedVolume(target, q)
	vols.tool, src.tool = analyticOrMeshedVolume(tool, q)
	vols.body, src.body = analyticOrMeshedVolume(body, q)
	return vols, src
}

// volumeTriple is the three volumes every acceptance bracket reads. They are kept together because
// they are measured in one pass and always travel as a set: passed one by one they made the bracket
// an eight-argument call in which nothing but position told the target from the tool.
type volumeTriple struct{ target, tool, body float64 }

// analyticOrMeshedVolume is BodyGeometryProperties' own choice made where the caller can SEE it: the
// analytic integral when the body admits one, the tessellation at q otherwise. It costs the same
// single integration either way, and it is what lets the brackets above know which number they hold.
func analyticOrMeshedVolume(b *topo.Body, q Quality) (float64, bool) {
	if props, ok := query.AnalyticGeometryProperties(b); ok {
		return props.Volume, true
	}
	return query.BodyGeometryProperties(b, q).Volume, false
}

// volumeOutOfBracket reports whether a result volume falls outside the Requicha two-sided
// bracket for op with tolerance tol (#1601): V(A∪B) ∈ [max(V(A),V(B)), V(A)+V(B)] and
// V(A∖B) ∈ [V(A)−V(B), V(A)]. The old one-sided checks let a join that fabricated material or a
// cut that removed too much ship silently. Intersect keeps only its upper bound V(A∩B) ≤
// min(V(A),V(B)): its Requicha lower bound needs V(A∪B) — a second boolean, too expensive for a
// guard — and is trivially ≥ 0 without it. Shared by the planar guard (a tight model-relative
// tol) and the curved analytic guard (a deficit-dominating relative tol, curvedExactGuarded).
func volumeOutOfBracket(op PartFeatureOperation, vols volumeTriple, tol float64) bool {
	targetVol, toolVol, bodyVol := vols.target, vols.tool, vols.body
	switch op {
	case Join:
		return bodyVol+tol < max(targetVol, toolVol) || bodyVol > targetVol+toolVol+tol
	case Cut:
		return bodyVol > targetVol+tol || bodyVol+tol < targetVol-toolVol
	case Intersect:
		return bodyVol > min(targetVol, toolVol)+tol
	}
	return false
}
