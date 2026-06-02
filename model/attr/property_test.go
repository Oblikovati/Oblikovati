// SPDX-License-Identifier: GPL-2.0-only

package attr

import (
	"testing"

	"github.com/Oblikovati/oblikovati/model/param"
)

func TestStandardPropertySetsExist(t *testing.T) {
	ps := NewPropertySets()
	for _, name := range []string{SummaryInformation, DocumentSummary, DesignTracking, UserDefined} {
		if _, ok := ps.Lookup(name); !ok {
			t.Errorf("standard set %q missing", name)
		}
	}
	if len(ps.Sets()) != 4 {
		t.Errorf("Sets count = %d, want 4 standard sets", len(ps.Sets()))
	}
}

func TestCustomPropertiesPersistAndAreQueryable(t *testing.T) {
	ps := NewPropertySets()
	ps.Set(SummaryInformation).Put("Author", StringValue("Vinicius"))
	ps.Set(UserDefined).Put("RoHS", BoolValue(true))

	back, err := DecodeProperties(ps.EncodeProperties())
	if err != nil {
		t.Fatalf("DecodeProperties: %v", err)
	}
	author, ok := mustSet(t, back, SummaryInformation).Property("Author")
	if !ok {
		t.Fatal("Author property lost on reload")
	}
	if s, _ := author.Value().Str(); s != "Vinicius" {
		t.Errorf("Author = %q, want Vinicius", s)
	}
	if rohs, ok := mustSet(t, back, UserDefined).Property("RoHS"); !ok {
		t.Error("custom RoHS property lost on reload")
	} else if b, _ := rohs.Value().Bool(); !b {
		t.Error("RoHS value wrong after reload")
	}
}

func mustSet(t *testing.T, ps *PropertySets, name string) *PropertySet {
	t.Helper()
	s, ok := ps.Lookup(name)
	if !ok {
		t.Fatalf("set %q missing", name)
	}
	return s
}

func TestExposedParameterAppearsAsCustomProperty(t *testing.T) {
	params := param.NewParameters()
	p, err := params.AddUserParameter("width", "25 mm")
	if err != nil {
		t.Fatalf("AddUserParameter: %v", err)
	}

	ps := NewPropertySets()
	prop := ps.ExposeParameter(p)

	custom, _ := ps.Lookup(UserDefined)
	got, ok := custom.Property("width")
	if !ok || got != prop {
		t.Fatal("exposed parameter did not appear in custom properties")
	}
	if src, ok := got.ExposedFromParameter(); !ok || src != "width" {
		t.Errorf("ExposedFromParameter = %q,%v, want width,true", src, ok)
	}
	if f, _ := got.Value().Float(); f != p.ModelValue() {
		t.Errorf("exposed value = %g, want parameter model value %g", f, p.ModelValue())
	}

	// Provenance survives a round trip.
	back, _ := DecodeProperties(ps.EncodeProperties())
	bs, _ := back.Lookup(UserDefined)
	rp, _ := bs.Property("width")
	if src, ok := rp.ExposedFromParameter(); !ok || src != "width" {
		t.Error("exposed-from provenance lost on reload")
	}
}

func TestPropertySetRemove(t *testing.T) {
	ps := NewPropertySets()
	s := ps.Set(UserDefined)
	s.Put("temp", IntValue(1))
	if !s.Remove("temp") || s.Count() != 0 || s.Remove("temp") {
		t.Error("PropertySet.Remove behavior wrong")
	}
}

func TestPropertyAccessors(t *testing.T) {
	s := newPropertySet("custom")
	if s.Name() != "custom" {
		t.Errorf("Name = %q", s.Name())
	}
	p := s.Put("k", IntValue(1))
	if p.Name() != "k" {
		t.Errorf("property Name = %q", p.Name())
	}
	// Put on an existing name updates in place.
	if s.Put("k", IntValue(2)) != p {
		t.Error("Put created a new property instead of updating")
	}
	p.SetValue(StringValue("v"))
	if got, _ := p.Value().Str(); got != "v" {
		t.Errorf("value after SetValue = %q, want v", got)
	}
	if src, ok := p.ExposedFromParameter(); ok || src != "" {
		t.Error("directly-authored property reports a parameter source")
	}
}

func TestDecodePropertiesRejectsBadValueType(t *testing.T) {
	// One set, one property whose value-type byte is invalid.
	var blob []byte
	blob = appendU32(blob, 1)       // nSets
	blob = appendString(blob, "s")  // set name
	blob = appendU32(blob, 1)       // nProps
	blob = appendString(blob, "p")  // prop name
	blob = appendString(blob, "")   // source
	blob = append(blob, byte(0xff)) // bogus value type
	if _, err := DecodeProperties(blob); err == nil {
		t.Error("DecodeProperties accepted an unknown value type")
	}
}

// appendU32 mirrors the codec's little-endian uint32 writer for test fixtures.
func appendU32(buf []byte, n uint32) []byte {
	return append(buf, byte(n), byte(n>>8), byte(n>>16), byte(n>>24))
}
