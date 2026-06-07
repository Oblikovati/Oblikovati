// SPDX-License-Identifier: GPL-2.0-only

package part21

import (
	"strings"
	"testing"
)

func TestFormatHelpersAndWriterBranches(t *testing.T) {
	if got := QuoteString("can't"); got != "'can''t'" {
		t.Fatalf("QuoteString = %q", got)
	}
	if got := FormatReal(42); got != "42." {
		t.Fatalf("FormatReal integer = %q", got)
	}
	if got := FormatReal(1e20); got != "1.e+20" {
		t.Fatalf("FormatReal exponent = %q", got)
	}
	if got := FormatList("A", "B"); got != "(A,B)" {
		t.Fatalf("FormatList = %q", got)
	}
	if FormatEnum("T") != ".T." || FormatBool(true) != ".T." || FormatBool(false) != ".F." {
		t.Fatal("enum/bool formatting mismatch")
	}
	w := NewWriter()
	if Ref(7) != "#7" {
		t.Fatal("Ref(7) did not format #7")
	}
	first := w.Add("DIRECTION", QuoteString(""), FormatList("0.", "0.", "1."))
	sharedA := w.AddShared("CARTESIAN_POINT", QuoteString(""), FormatList("0.", "0.", "0."))
	sharedB := w.AddShared("CARTESIAN_POINT", QuoteString(""), FormatList("0.", "0.", "0."))
	raw := w.AddRaw("(NAMED_UNIT(*)SI_UNIT(.MILLI.,.METRE.)LENGTH_UNIT())")
	if first != 1 || sharedA != 2 || sharedB != sharedA || raw != 3 {
		t.Fatalf("writer ids = %d %d %d %d", first, sharedA, sharedB, raw)
	}
	out := string(w.Emit(Header{Description: []string{"test"}, ImplementationLevel: "2;1", Name: "x.step", SchemaIdentifiers: []string{"CONFIG_CONTROL_DESIGN"}}))
	for _, want := range []string{"HEADER;", "#1=DIRECTION", "#2=CARTESIAN_POINT", "#3=(NAMED_UNIT"} {
		if !strings.Contains(out, want) {
			t.Fatalf("writer output missing %q:\n%s", want, out)
		}
	}
}

func TestValueAccessorsAndGraphQueries(t *testing.T) {
	pos := Token{Line: 4, Column: 2}
	if got, err := (Value{Kind: ValInt, Int: 5}).AsFloat(); err != nil || got != 5 {
		t.Fatalf("AsFloat(int) = %g, %v", got, err)
	}
	if got, err := (Value{Kind: ValReal, Real: 1.25}).AsFloat(); err != nil || got != 1.25 {
		t.Fatalf("AsFloat(real) = %g, %v", got, err)
	}
	if got, err := (Value{Kind: ValInt, Int: 5}).AsInt(); err != nil || got != 5 {
		t.Fatalf("AsInt = %d, %v", got, err)
	}
	if got, err := (Value{Kind: ValString, Str: "abc"}).AsString(); err != nil || got != "abc" {
		t.Fatalf("AsString = %q, %v", got, err)
	}
	if got, err := (Value{Kind: ValEnum, Enum: "T"}).AsEnum(); err != nil || got != "T" {
		t.Fatalf("AsEnum = %q, %v", got, err)
	}
	if got, err := (Value{Kind: ValEnum, Enum: "T"}).AsBool(); err != nil || !got {
		t.Fatalf("AsBool(.T.) = %v, %v", got, err)
	}
	if _, err := (Value{Kind: ValEnum, Enum: "U", position: pos}).AsBool(); err == nil {
		t.Fatal("AsBool accepted non-boolean enum")
	}
	if args := (Value{Kind: ValTyped, Keyword: "LENGTH_MEASURE", List: []Value{{Kind: ValReal, Real: 1.5}}}).TypedArgs(); len(args) != 1 {
		t.Fatalf("TypedArgs length = %d", len(args))
	}
	if kw := (Value{Kind: ValTyped, Keyword: "LENGTH_MEASURE"}).TypedKeyword(); kw != "LENGTH_MEASURE" {
		t.Fatalf("TypedKeyword = %q", kw)
	}
	if (Value{Kind: ValInt}).TypedArgs() != nil || (Value{Kind: ValInt}).TypedKeyword() != "" {
		t.Fatal("non-typed value returned typed metadata")
	}
	if _, err := (Value{Kind: ValReal, position: pos}).AsRef(); err == nil || !strings.Contains(err.Error(), "expected reference") {
		t.Fatalf("AsRef wrong-kind error = %v", err)
	}
	if got := ValueKind(99).String(); got != "99" {
		t.Fatalf("unknown ValueKind string = %q", got)
	}
	for _, tc := range []struct {
		kind ValueKind
		want string
	}{
		{ValRef, "ref"}, {ValInt, "int"}, {ValReal, "real"}, {ValString, "string"},
		{ValEnum, "enum"}, {ValList, "list"}, {ValNull, "null"}, {ValDerived, "derived"}, {ValTyped, "typed"},
	} {
		if got := tc.kind.String(); got != tc.want {
			t.Fatalf("%d.String() = %q, want %q", tc.kind, got, tc.want)
		}
	}
	for _, tc := range []struct {
		kind TokenKind
		want string
	}{
		{TokKeyword, "keyword"}, {TokRef, "ref"}, {TokString, "string"}, {TokInt, "int"},
		{TokReal, "real"}, {TokEnum, "enum"}, {TokLParen, "("}, {TokRParen, ")"},
		{TokComma, ","}, {TokEquals, "="}, {TokSemicolon, ";"}, {TokStar, "*"},
		{TokDollar, "$"}, {TokEOF, "eof"}, {TokenKind(99), "kind(99)"},
	} {
		if got := tc.kind.String(); got != tc.want {
			t.Fatalf("token %d.String() = %q, want %q", tc.kind, got, tc.want)
		}
	}

	f, err := Parse([]byte(wrapData("#1=DIRECTION('',(0.,0.,1.));\n#2=DIRECTION('',(1.,0.,0.));\n#3=CARTESIAN_POINT('',(0.,0.,0.));\n#4=MEASURE_REPRESENTATION_ITEM('x',LENGTH_MEASURE(25),.MM.);")))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	ids := f.Graph.IDs()
	if len(ids) != 4 || ids[0] != 1 || ids[3] != 4 {
		t.Fatalf("IDs = %v", ids)
	}
	dirs := f.Graph.EntitiesOfType("DIRECTION")
	if len(dirs) != 2 || dirs[0].ID != 1 || dirs[1].ID != 2 {
		t.Fatalf("EntitiesOfType(DIRECTION) = %#v", dirs)
	}
	item, err := f.Graph.Lookup(4)
	if err != nil {
		t.Fatalf("Lookup #4: %v", err)
	}
	if item.Params[1].TypedKeyword() != "LENGTH_MEASURE" {
		t.Fatalf("typed keyword = %q", item.Params[1].TypedKeyword())
	}
}
