// SPDX-License-Identifier: GPL-2.0-only

package blenderxml

import (
	"bytes"
	"testing"
)

// sample mirrors the shape of a Blender theme file: nested single-purpose elements with
// all data in attributes.
const sample = `<bpy>
  <Theme>
    <user_interface>
      <ThemeUserInterface editor_border="#161616" panel_back="#1b1b1bff">
        <wcol_regular>
          <ThemeWidgetColors inner="#242424ff" text="#c5c5c5"></ThemeWidgetColors>
        </wcol_regular>
      </ThemeUserInterface>
    </user_interface>
  </Theme>
</bpy>
`

func mustParse(t *testing.T, data string) *Node {
	t.Helper()
	root, err := Parse([]byte(data))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return root
}

func TestParseFindAttr(t *testing.T) {
	root := mustParse(t, sample)
	wcol := root.Find("Theme", "user_interface", "ThemeUserInterface", "wcol_regular", "ThemeWidgetColors")
	if wcol == nil {
		t.Fatal("Find returned nil for a path present in the document")
	}
	if v, ok := wcol.Attr("inner"); !ok || v != "#242424ff" {
		t.Errorf("Attr(inner) = (%q,%v), want (\"#242424ff\",true)", v, ok)
	}
	if root.Find("Theme", "no_such_section") != nil {
		t.Error("Find on a missing path should return nil")
	}
	if _, ok := root.Find("Theme").Attr("missing"); ok {
		t.Error("Attr on a missing attribute should report absent")
	}
}

func TestMarshalRoundTripPreservesUnmappedContent(t *testing.T) {
	root := mustParse(t, sample)
	out, err := root.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	again := mustParse(t, string(out))
	ui := again.Find("Theme", "user_interface", "ThemeUserInterface")
	if v, _ := ui.Attr("panel_back"); v != "#1b1b1bff" {
		t.Errorf("panel_back lost in round-trip: %q", v)
	}
	if !bytes.HasSuffix(out, []byte("\n")) {
		t.Error("Marshal output must end with a newline")
	}
}

func TestSetAttrOverwritesInPlaceAndAppends(t *testing.T) {
	root := mustParse(t, sample)
	ui := root.Find("Theme", "user_interface", "ThemeUserInterface")
	ui.SetAttr("editor_border", "#000000")
	if ui.Attrs[0].Name.Local != "editor_border" || ui.Attrs[0].Value != "#000000" {
		t.Errorf("SetAttr did not overwrite in place: %+v", ui.Attrs[0])
	}
	ui.SetAttr("brand_new", "#123456")
	if v, ok := ui.Attr("brand_new"); !ok || v != "#123456" {
		t.Errorf("SetAttr did not append a new attribute: (%q,%v)", v, ok)
	}
}

func TestCloneIsDeeplyIndependent(t *testing.T) {
	root := mustParse(t, sample)
	cp := root.Clone()
	cp.Find("Theme", "user_interface", "ThemeUserInterface").SetAttr("panel_back", "#ffffffff")
	orig, _ := root.Find("Theme", "user_interface", "ThemeUserInterface").Attr("panel_back")
	if orig != "#1b1b1bff" {
		t.Errorf("editing a clone mutated the original: %q", orig)
	}
}

func TestAppendAndRemoveChild(t *testing.T) {
	root := mustParse(t, sample)
	root.AppendChild(NewElement("oblikovati_tokens"))
	if root.Find("oblikovati_tokens") == nil {
		t.Fatal("AppendChild did not attach the new element")
	}
	root.RemoveChild("oblikovati_tokens")
	if root.Find("oblikovati_tokens") != nil {
		t.Error("RemoveChild did not delete the element")
	}
}

func TestParseRejectsMalformedXML(t *testing.T) {
	if _, err := Parse([]byte("<bpy><Theme></bpy>")); err == nil {
		t.Error("Parse of mismatched tags should fail")
	}
}
