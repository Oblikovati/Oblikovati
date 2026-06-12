// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"strings"
	"testing"
)

func TestExportXMLCoversUserParametersOnly(t *testing.T) {
	ps := NewParameters()
	num, _ := ps.AddUserParameter("len", "10 mm")
	num.Comment, num.IsKey = "the length", true
	_, _ = ps.AddTextUserParameter("material", "steel")
	_, _ = ps.AddBooleanUserParameter("vented", true)
	_, _ = ps.AddModelParameter("d0", "2 mm")

	xml, err := ps.ExportXML()
	if err != nil {
		t.Fatalf("ExportXML: %v", err)
	}
	for _, want := range []string{
		`name="len"`, `expression="10 mm"`, `comment="the length"`, `isKey="true"`,
		`name="material"`, `valueType="text"`, `text="steel"`,
		`name="vented"`, `valueType="boolean"`, `bool="true"`,
	} {
		if !strings.Contains(xml, want) {
			t.Errorf("export missing %s:\n%s", want, xml)
		}
	}
	if strings.Contains(xml, "d0") {
		t.Error("model parameters must not export — they are feature-owned")
	}
}

func TestImportXMLRoundTripAddsAndUpdates(t *testing.T) {
	src := NewParameters()
	num, _ := src.AddUserParameter("len", "10 mm")
	num.ExposedAsProperty = true
	_, _ = src.AddTextUserParameter("material", "steel")
	xml, _ := src.ExportXML()

	// A fresh collection imports everything as new.
	dst := NewParameters()
	added, updated, err := dst.ImportXML(xml)
	if err != nil {
		t.Fatalf("ImportXML: %v", err)
	}
	if added != 2 || updated != 0 {
		t.Errorf("import counts = %d/%d, want 2 added", added, updated)
	}
	rl, _ := dst.ByName("len")
	if rl == nil || rl.Expression() != "10 mm" || !rl.ExposedAsProperty {
		t.Errorf("len not round-tripped: %+v", rl)
	}
	rt, _ := dst.ByName("material")
	if rt == nil || !rt.IsText() || rt.Text() != "steel" {
		t.Errorf("material not round-tripped: %+v", rt)
	}

	// A second import of an edited set updates in place.
	edited := strings.Replace(xml, "10 mm", "12 mm", 1)
	added, updated, err = dst.ImportXML(edited)
	if err != nil {
		t.Fatalf("re-import: %v", err)
	}
	if added != 0 || updated != 2 {
		t.Errorf("re-import counts = %d/%d, want 2 updated", added, updated)
	}
	if rl.Expression() != "12 mm" {
		t.Errorf("len expression = %q, want the imported 12 mm", rl.Expression())
	}
}

func TestImportXMLRejectsBadSets(t *testing.T) {
	for name, doc := range map[string]string{
		"not xml":                    `{"name":"len"}`,
		"missing name":               `<parameters><parameter expression="10 mm"/></parameters>`,
		"unknown value type":         `<parameters><parameter name="x" valueType="blob"/></parameters>`,
		"numeric without expression": `<parameters><parameter name="x"/></parameters>`,
		"duplicate name":             `<parameters><parameter name="x" expression="1 mm"/><parameter name="x" expression="2 mm"/></parameters>`,
	} {
		ps := NewParameters()
		if _, _, err := ps.ImportXML(doc); err == nil {
			t.Errorf("%s: import must be rejected", name)
		}
		if ps.Count() != 0 {
			t.Errorf("%s: structural rejection must not create parameters", name)
		}
	}
}

func TestImportXMLFlavorMismatchErrors(t *testing.T) {
	ps := NewParameters()
	_, _ = ps.AddUserParameter("len", "10 mm")
	doc := `<parameters><parameter name="len" valueType="text" text="oops"/></parameters>`
	if _, _, err := ps.ImportXML(doc); err == nil || !strings.Contains(err.Error(), "len") {
		t.Errorf("flavor mismatch err = %v, want a rejection naming len", err)
	}
}
