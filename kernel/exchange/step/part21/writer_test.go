// SPDX-License-Identifier: GPL-2.0-only

package part21

import "testing"

func TestWriterAllocatesMonotonicIDs(t *testing.T) {
	w := NewWriter()
	a := w.Add("DIRECTION", QuoteString(""), FormatList(FormatReal(0), FormatReal(0), FormatReal(1)))
	b := w.Add("CARTESIAN_POINT", QuoteString(""), FormatList(FormatReal(1), FormatReal(2), FormatReal(3)))
	if a != 1 || b != 2 {
		t.Errorf("ids = %d,%d, want 1,2", a, b)
	}
}

func TestWriterSharesIdenticalStatements(t *testing.T) {
	w := NewWriter()
	origin := FormatList(FormatReal(0), FormatReal(0), FormatReal(0))
	a := w.AddShared("CARTESIAN_POINT", QuoteString(""), origin)
	b := w.AddShared("CARTESIAN_POINT", QuoteString(""), origin)
	if a != b {
		t.Errorf("identical shared points got ids %d and %d, want the same", a, b)
	}
}

func TestWriterEmitRoundTrips(t *testing.T) {
	w := NewWriter()
	w.Add("DIRECTION", QuoteString("z"), FormatList(FormatReal(0), FormatReal(0), FormatReal(1)))
	h := Header{
		Description: []string{"x"}, ImplementationLevel: "2;1",
		SchemaIdentifiers: []string{"CONFIG_CONTROL_DESIGN"},
	}
	out := w.Emit(h)
	f, err := Parse(out)
	if err != nil {
		t.Fatalf("emitted file does not re-parse: %v\n%s", err, out)
	}
	if f.Graph.Len() != 1 {
		t.Errorf("re-parsed %d entities, want 1", f.Graph.Len())
	}
}

func TestWriterEmitIsByteStable(t *testing.T) {
	build := func() []byte {
		w := NewWriter()
		w.AddShared("CARTESIAN_POINT", QuoteString(""), FormatList(FormatReal(0), FormatReal(0), FormatReal(0)))
		w.Add("DIRECTION", QuoteString(""), FormatList(FormatReal(0), FormatReal(0), FormatReal(1)))
		return w.Emit(Header{SchemaIdentifiers: []string{"CONFIG_CONTROL_DESIGN"}})
	}
	first := string(build())
	second := string(build())
	if first != second {
		t.Error("emit is not byte-stable across runs")
	}
}

func TestFormatReal(t *testing.T) {
	cases := map[float64]string{0: "0.", 1: "1.", 3.14: "3.14", -2.5: "-2.5"}
	for in, want := range cases {
		if got := FormatReal(in); got != want {
			t.Errorf("FormatReal(%v) = %q, want %q", in, got, want)
		}
	}
}
