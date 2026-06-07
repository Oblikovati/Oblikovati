// SPDX-License-Identifier: GPL-2.0-only

package schema

import "testing"

func TestDetectProtocols(t *testing.T) {
	cases := []struct {
		ids  []string
		want ApProtocol
	}{
		{[]string{"CONFIG_CONTROL_DESIGN"}, AP203},
		{[]string{"AUTOMOTIVE_DESIGN { 1 0 10303 214 1 1 1 1 }"}, AP214},
		{[]string{"AP242_MANAGED_MODEL_BASED_3D_ENGINEERING_MIM_LF { 1 0 10303 442 1 1 4 }"}, AP242},
		{[]string{"SOMETHING_ELSE"}, ApUnknown},
		{nil, ApUnknown},
	}
	for _, c := range cases {
		if got := Detect(c.ids); got != c.want {
			t.Errorf("Detect(%v) = %s, want %s", c.ids, got, c.want)
		}
	}
}

func TestApProtocolString(t *testing.T) {
	for _, tc := range []struct {
		ap   ApProtocol
		want string
	}{
		{AP203, "AP203"},
		{AP214, "AP214"},
		{AP242, "AP242"},
		{ApUnknown, "unknown"},
	} {
		if got := tc.ap.String(); got != tc.want {
			t.Fatalf("%d.String() = %q, want %q", tc.ap, got, tc.want)
		}
	}
}

func TestSchemaIdentifierRoundTrips(t *testing.T) {
	for _, ap := range []ApProtocol{AP203, AP214, AP242} {
		id, err := SchemaIdentifier(ap)
		if err != nil {
			t.Fatalf("SchemaIdentifier(%s): %v", ap, err)
		}
		if got := Detect([]string{id}); got != ap {
			t.Errorf("round trip %s → %q → %s", ap, id, got)
		}
	}
}

func TestSchemaIdentifierUnknownErrors(t *testing.T) {
	if _, err := SchemaIdentifier(ApUnknown); err == nil {
		t.Error("SchemaIdentifier(ApUnknown) should error")
	}
}
