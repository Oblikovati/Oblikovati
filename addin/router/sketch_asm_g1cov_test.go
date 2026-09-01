// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// G1 router-coverage tests (M40): drive the still-untested error/edge branches of the
// sketch-reference, assembly-feature, views, parameters and drawing handlers converted to
// the typed router. Helpers are prefixed sacov to stay cluster-unique against the sibling
// skcov*/asmcov*/dpcov* fixtures; the shared harness (call/mustJSON/seededSession/...) is
// reused, never redefined.

// sacovMethodArgs pairs a wire method with a JSON argument string for the table-driven
// error-path tests below.
type sacovMethodArgs struct {
	method string
	args   string
}

// sacovAllReject asserts every (method,args) pair returns a handler error.
func sacovAllReject(t *testing.T, r *Router, s *app.Session, cases []sacovMethodArgs) {
	t.Helper()
	for _, c := range cases {
		if _, err := r.Handle(s, c.method, []byte(c.args)); err == nil {
			t.Errorf("%s(%s) must fail", c.method, c.args)
		}
	}
}

// --- parameters.go: get/set/add error branches (unknown name, missing field, bad expr) ---

func TestSacovParametersErrorBranches(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	sacovAllReject(t, r, s, []sacovMethodArgs{
		{"parameters.get", `{"name":"ghost"}`},                     // getParameter: ByName miss
		{"parameters.set", `{"name":"ghost","expression":"1 cm"}`}, // setParameter: ByName miss
		{"parameters.add", `{"name":"","expression":"1 cm"}`},      // requireNameExpr: empty name
		{"parameters.set", `{"name":"width","expression":""}`},     // requireNameExpr: empty expr
		{"parameters.add", `{"name":"bad","expression":"1 +"}`},    // AddUserParameter: parse error
		{"parameters.set", `{"name":"width","expression":"1 +"}`},  // SetExpression: parse error
	})
}

// --- views.go: unknown document + out-of-range index error branches ---

func TestSacovViewsUnknownDocument(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	sacovAllReject(t, r, s, []sacovMethodArgs{
		{"views.list", `{"document":999999}`},
		{"views.add", `{"document":999999,"name":"X"}`},
		{"views.activate", `{"document":999999,"index":0}`},
		{"views.close", `{"document":999999,"index":0}`},
		{"views.rename", `{"document":999999,"index":0,"name":"X"}`},
		{"views.getLayout", `{"document":999999}`},
		{"views.setLayout", `{"document":999999,"layout":1}`},
	})
}

func TestSacovViewsOutOfRangeIndex(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	sacovAllReject(t, r, s, []sacovMethodArgs{
		{"views.activate", `{"index":99}`}, // Views().Activate out of range
		{"views.close", `{"index":99}`},    // Views().Close out of range
		{"views.rename", `{"index":99,"name":"X"}`},
	})
}

// --- sketch_reference.go: vertical-justify mapping (parseVJustify/vAlignString) ---

// TestSacovAddTextVerticalAlignments exercises the parseVJustify lower/upper branches and its
// default (an unrecognised spelling maps to baseline), read back through vAlignString.
func TestSacovAddTextVerticalAlignments(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	want := map[string]string{"lower": "lower", "upper": "upper", "garbage": "baseline"}
	for in, exp := range want {
		if got := sacovTextVAlign(t, r, s, in); got != exp {
			t.Errorf("vJustify %q rendered VAlign %q, want %q", in, got, exp)
		}
	}
}

// sacovTextVAlign adds a text entity with the given vJustify and returns its read-back VAlign.
func sacovTextVAlign(t *testing.T, r *Router, s *app.Session, vj string) string {
	t.Helper()
	var added wire.AddEntityIDResult
	args := fmt.Sprintf(`{"sketchIndex":0,"anchor":[1,1],"text":"T","height":"5 mm","vJustify":%q}`, vj)
	call(t, r, s, "sketch.addText", args, &added)
	var got wire.SketchTextResult
	call(t, r, s, "sketch.getText", fmt.Sprintf(`{"sketchIndex":0,"entityId":%d}`, added.EntityID), &got)
	return string(got.Style.VAlign)
}

// sacovAddText adds a plain text entity to seeded sketch 0 and returns its id.
func sacovAddText(t *testing.T, r *Router, s *app.Session) uint64 {
	t.Helper()
	var added wire.AddEntityIDResult
	call(t, r, s, "sketch.addText", `{"sketchIndex":0,"anchor":[1,1],"text":"T","height":"5 mm"}`, &added)
	return added.EntityID
}

// TestSacovEditTextAllFields drives applyTextLengths (height/fontSize/rotation) and the
// applyTextEdit font + vJustify branches in one partial edit.
func TestSacovEditTextAllFields(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	id := sacovAddText(t, r, s)
	var edited wire.SketchTextResult
	args := fmt.Sprintf(`{"sketchIndex":0,"entityId":%d,"height":"8 mm","fontSize":"6 mm","rotation":"30 deg","font":"Liberation Sans","vJustify":"upper"}`, id)
	call(t, r, s, "sketch.editText", args, &edited)
	if edited.Style.Height == 0 || edited.Style.FontSize == 0 || edited.Style.Rotation == 0 {
		t.Fatalf("edit did not apply the lengths: %+v", edited.Style)
	}
	if string(edited.Style.VAlign) != "upper" || edited.Style.Family != "Liberation Sans" {
		t.Errorf("edited family/valign = %+v, want upper/Liberation Sans", edited.Style)
	}
}

// TestSacovSketchTextImageErrors pins the add/edit/get text and add-image error branches:
// bad unit-bearing metrics, bad anchor, bad sketch index, and missing/mistyped entity refs.
func TestSacovSketchTextImageErrors(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	id := sacovAddText(t, r, s)
	lineID := skcovLineIDs(t, r, s)[0]
	sacovAllReject(t, r, s, sacovTextErrorCases(id, lineID))
	sacovAllReject(t, r, s, sacovImageErrorCases())
}

// sacovTextErrorCases enumerates the addText/getText/editText error inputs.
func sacovTextErrorCases(id, lineID uint64) []sacovMethodArgs {
	return []sacovMethodArgs{
		{"sketch.addText", `{"sketchIndex":0,"anchor":[1,1],"text":"T","height":"1 +"}`},                   // textMetrics height
		{"sketch.addText", `{"sketchIndex":0,"anchor":[1,1],"text":"T","height":"5 mm","rotation":"1 +"}`}, // rotation
		{"sketch.addText", `{"sketchIndex":0,"anchor":[1,1],"text":"T","height":"5 mm","fontSize":"1 +"}`}, // fontSize
		{"sketch.addText", `{"sketchIndex":0,"anchor":[1],"text":"T","height":"5 mm"}`},                    // anchor
		{"sketch.addText", `{"sketchIndex":9,"anchor":[1,1],"text":"T","height":"5 mm"}`},                  // sketchAtIndex
		{"sketch.getText", `{"sketchIndex":0,"entityId":999999}`},                                          // textBoxRef: missing
		{"sketch.getText", fmt.Sprintf(`{"sketchIndex":0,"entityId":%d}`, lineID)},                         // textBoxRef: not a text box
		{"sketch.editText", `{"sketchIndex":0,"entityId":999999}`},                                         // editText: missing
		{"sketch.editText", fmt.Sprintf(`{"sketchIndex":0,"entityId":%d,"height":"1 +"}`, id)},             // applyTextLengths height
		{"sketch.editText", fmt.Sprintf(`{"sketchIndex":0,"entityId":%d,"fontSize":"1 +"}`, id)},           // fontSize
		{"sketch.editText", fmt.Sprintf(`{"sketchIndex":0,"entityId":%d,"rotation":"1 +"}`, id)},           // rotation
	}
}

// sacovImageErrorCases enumerates the addImage error inputs (imageMetrics + anchor + index).
func sacovImageErrorCases() []sacovMethodArgs {
	return []sacovMethodArgs{
		{"sketch.addImage", `{"sketchIndex":0,"ref":"x","anchor":[0,0],"width":"1 +","height":"1 cm"}`},                   // width
		{"sketch.addImage", `{"sketchIndex":0,"ref":"x","anchor":[0,0],"width":"1 cm","height":"1 +"}`},                   // height
		{"sketch.addImage", `{"sketchIndex":0,"ref":"x","anchor":[0,0],"width":"1 cm","height":"1 cm","rotation":"1 +"}`}, // rotation
		{"sketch.addImage", `{"sketchIndex":0,"ref":"x","anchor":[0],"width":"1 cm","height":"1 cm"}`},                    // anchor
		{"sketch.addImage", `{"sketchIndex":9,"ref":"x","anchor":[0,0],"width":"1 cm","height":"1 cm"}`},                  // sketchAtIndex
	}
}

// TestSacovAddSketchImageSucceeds drives the addSketchImage happy path + imageMetrics with a
// rotation, so the placed entity carries an id (Ref is just a package-store string — no I/O).
func TestSacovAddSketchImageSucceeds(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	var res wire.AddSketchImageResult
	call(t, r, s, "sketch.addImage",
		`{"sketchIndex":0,"ref":"pkg://logo","anchor":[0,0],"width":"4 cm","height":"2 cm","rotation":"15 deg","opacity":0.5}`, &res)
	if res.EntityID == 0 {
		t.Fatal("addImage returned no entity id")
	}
}

// --- assembly_features.go: addProxyCut/addExtrude/add error branches (cutOperation, source,
// sketch index, non-positive distance) ---

func TestSacovAssemblyFeatureAddErrors(t *testing.T) {
	t.Parallel()
	r, s, _, _ := assemblySessionWithBoxes(t, 0, 5)
	sacovAllReject(t, r, s, []sacovMethodArgs{
		{"assemblyFeatures.addProxyCut", `{"source":123,"operation":"bogus"}`},                                      // cutOperation
		{"assemblyFeatures.addProxyCut", `{"source":999999,"operation":"difference"}`},                              // no occurrence
		{"assemblyFeatures.addExtrude", `{"sketchIndex":0,"profileIndex":0,"distance":1,"operation":"bogus"}`},      // cutOperation
		{"assemblyFeatures.addExtrude", `{"sketchIndex":9,"profileIndex":0,"distance":1,"operation":"difference"}`}, // sketchAtIndex
		{"assemblyFeatures.add", `{"toolMin":[0,0,0],"toolMax":[1,1,1],"operation":"bogus"}`},                       // assemblyCutFromArgs
	})
}

// TestSacovAssemblyExtrudeRejectsNonPositiveDistance reaches the distance<=0 guard, which
// needs a real assembly sketch profile before the distance is validated.
func TestSacovAssemblyExtrudeRejectsNonPositiveDistance(t *testing.T) {
	t.Parallel()
	r, s, _, _ := assemblySessionWithBoxes(t, 0)
	var sk wire.CreateSketchResult
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &sk)
	call(t, r, s, "sketch.rectangle",
		fmt.Sprintf(`{"sketchIndex":%d,"width":"0.5 cm","height":"1 cm"}`, sk.SketchIndex), &wire.SketchRectangleResult{})
	args := fmt.Sprintf(`{"sketchIndex":%d,"profileIndex":0,"distance":0,"operation":"difference"}`, sk.SketchIndex)
	if _, err := r.Handle(s, "assemblyFeatures.addExtrude", []byte(args)); err == nil {
		t.Error("a non-positive assembly extrude distance must be rejected")
	}
}

// --- drawing.go: sheet + title-block + export error branches, and the activeDrawing guard ---

func TestSacovDrawingErrorBranches(t *testing.T) {
	t.Parallel()
	r, s := drawingSession(t)
	sacovAllReject(t, r, s, []sacovMethodArgs{
		{"drawing.addSheet", `{"size":"bogus"}`},          // sheetSpecOf: size
		{"drawing.addSheet", `{"orientation":"bogus"}`},   // sheetSpecOf: orientation
		{"drawing.removeSheet", `{"name":"Ghost"}`},       // Sheets().Remove
		{"drawing.setActiveSheet", `{"name":"Ghost"}`},    // Sheets().SetActive
		{"drawing.titleBlockFields", `{"sheet":"Ghost"}`}, // sheetByNameOrActive
		{"drawing.exportDXF", `{"path":""}`},              // empty export path (no I/O)
	})
}

func TestSacovDrawingHandlersRequireDrawingDoc(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t) // active part, not a drawing → activeDrawing guard
	sacovAllReject(t, r, s, []sacovMethodArgs{
		{"drawing.listSheets", `{}`},
		{"drawingViews.list", `{}`},
	})
}

// --- drawing_views.go: view style + section success, and style/section/delete errors ---

// TestSacovDrawingViewStyleAndSection covers parseViewStyle's valid non-default branch
// ("shaded") and the previously-untested drawingViewsAddSection happy path.
func TestSacovDrawingViewStyleAndSection(t *testing.T) {
	t.Parallel()
	r, s := drawingViewSession(t)
	var base wire.ViewResult
	call(t, r, s, "drawingViews.addBase",
		`{"name":"FRONT","orientation":"front","style":"shaded","scale":2,"centerXmm":120,"centerYmm":100}`, &base)
	if base.View.Style != "shaded" {
		t.Fatalf("base view style = %q, want shaded", base.View.Style)
	}
	var sec wire.ViewResult
	call(t, r, s, "drawingViews.addSection",
		`{"name":"SEC","parentView":"FRONT","x1":80,"y1":100,"x2":160,"y2":100,"centerXmm":120,"centerYmm":260}`, &sec)
	if sec.View.Type != "section" || sec.View.BaseView != "FRONT" {
		t.Fatalf("section view = %+v, want a section off FRONT", sec.View)
	}
}

func TestSacovDrawingViewErrorBranches(t *testing.T) {
	t.Parallel()
	r, s := drawingViewSession(t)
	call(t, r, s, "drawingViews.addBase", `{"name":"FRONT","scale":2,"centerXmm":120,"centerYmm":100}`, nil)
	sacovAllReject(t, r, s, []sacovMethodArgs{
		{"drawingViews.addBase", `{"style":"bogus"}`},                                    // parseViewStyle error
		{"drawingViews.addSection", `{"parentView":"NOPE","x1":0,"y1":0,"x2":1,"y2":1}`}, // AddSection: unknown parent
		{"drawingViews.delete", `{"name":"Ghost"}`},                                      // Remove: unknown view
	})
}
