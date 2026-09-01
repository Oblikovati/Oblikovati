// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// writeLINFixture writes a .lin fixture with DASHDOT and BORDER definitions.
func writeLINFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "styles.lin")
	src := "*DASHDOT,dash dot\nA,.5,-.25,0,-.25\n*BORDER,border\nA,.5,-.25,.5,-.25,0,-.25\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

// sketchSession builds a part session with one empty sketch.
func sketchSession(t *testing.T) (*Router, *app.Session) {
	t.Helper()
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	return r, s
}

// TestSketchCustomLineTypeLoadGetPersist drives the full path: load DASHDOT, read it
// back, and confirm sketch.get reports lineType=custom.
func TestSketchCustomLineTypeLoadGetPersist(t *testing.T) {
	t.Parallel()
	r, s := sketchSession(t)
	lin := writeLINFixture(t)

	var set wire.SketchCustomLineTypeResult
	call(t, r, s, "sketch.setCustomLineType",
		fmt.Sprintf(`{"sketchIndex":0,"fullFileName":%q,"lineTypeName":"DASHDOT"}`, lin), &set)
	if !set.Loaded || set.LineTypeName != "DASHDOT" || len(set.Pattern) != 4 {
		t.Fatalf("set result = %+v, want loaded DASHDOT with 4 elements", set)
	}

	var got wire.SketchCustomLineTypeResult
	call(t, r, s, "sketch.getCustomLineType", `{"sketchIndex":0}`, &got)
	if !got.Loaded || got.LineTypeName != "DASHDOT" || got.FullFileName != lin {
		t.Errorf("get result = %+v, want DASHDOT from %q", got, lin)
	}

	var info wire.SketchInfo
	call(t, r, s, "sketch.get", `{"sketchIndex":0}`, &info)
	if info.LineType != "custom" {
		t.Errorf("sketch lineType = %q, want custom", info.LineType)
	}
}

// TestSketchCustomLineTypeReplaceExisting pins the replace contract: re-loading the
// same name fails without the flag and succeeds with it.
func TestSketchCustomLineTypeReplaceExisting(t *testing.T) {
	t.Parallel()
	r, s := sketchSession(t)
	lin := writeLINFixture(t)
	args := fmt.Sprintf(`{"sketchIndex":0,"fullFileName":%q,"lineTypeName":"DASHDOT"}`, lin)
	call(t, r, s, "sketch.setCustomLineType", args, &wire.SketchCustomLineTypeResult{})

	if _, err := r.Handle(s, "sketch.setCustomLineType", []byte(args)); err == nil {
		t.Error("re-loading DASHDOT without replaceExisting must fail")
	}
	var res wire.SketchCustomLineTypeResult
	call(t, r, s, "sketch.setCustomLineType",
		fmt.Sprintf(`{"sketchIndex":0,"fullFileName":%q,"lineTypeName":"DASHDOT","replaceExisting":true}`, lin), &res)
	if !res.Loaded {
		t.Errorf("replaceExisting reload = %+v, want loaded", res)
	}
}

// TestSketchCustomLineTypeErrors covers the failure paths: no sketch, missing file,
// unknown definition name, empty name, and unloaded get.
func TestSketchCustomLineTypeErrors(t *testing.T) {
	t.Parallel()
	r, s := sketchSession(t)
	lin := writeLINFixture(t)
	bad := []string{
		fmt.Sprintf(`{"sketchIndex":9,"fullFileName":%q,"lineTypeName":"DASHDOT"}`, lin),
		`{"sketchIndex":0,"fullFileName":"/nonexistent/x.lin","lineTypeName":"DASHDOT"}`,
		fmt.Sprintf(`{"sketchIndex":0,"fullFileName":%q,"lineTypeName":"NOPE"}`, lin),
		fmt.Sprintf(`{"sketchIndex":0,"fullFileName":%q,"lineTypeName":""}`, lin),
	}
	for _, b := range bad {
		if _, err := r.Handle(s, "sketch.setCustomLineType", []byte(b)); err == nil {
			t.Errorf("expected error for %s", b)
		}
	}
	var got wire.SketchCustomLineTypeResult
	call(t, r, s, "sketch.getCustomLineType", `{"sketchIndex":0}`, &got)
	if got.Loaded {
		t.Errorf("get with nothing loaded = %+v, want loaded=false", got)
	}
}
