// SPDX-License-Identifier: GPL-2.0-only

package attr

import (
	"bytes"
	"testing"
)

// TestValueTypePersistBytesAreStable pins the on-disk type byte for every value kind. ValueType is now
// an alias of the int32 api/types.ValueType (whose frozen Inventor numbers are large, e.g. Boolean =
// 14597), but the attribute codec must keep writing the compact 0..4 codes so existing .obk metadata
// stays byte-identical (#1501). This guards that mapping: a regression (e.g. reverting to byte(v.typ),
// which would write 5 for Boolean) turns it red.
func TestValueTypePersistBytesAreStable(t *testing.T) {
	cases := []struct {
		v        Value
		wantByte byte
	}{
		{BoolValue(true), 0},
		{IntValue(7), 1},
		{FloatValue(2.5), 2},
		{StringValue("x"), 3},
		{BytesValue([]byte{9}), 4},
	}
	for _, c := range cases {
		enc := encodeValue(nil, c.v)
		if len(enc) == 0 || enc[0] != c.wantByte {
			t.Errorf("%v: on-disk type byte = %v, want %d", c.v, enc, c.wantByte)
		}
		dv, err := decodeValue(bytes.NewReader(enc))
		if err != nil {
			t.Errorf("%v: decode: %v", c.v, err)
			continue
		}
		if dv.Type() != c.v.Type() {
			t.Errorf("%v: round-trip type = %v, want %v", c.v, dv.Type(), c.v.Type())
		}
	}
	// An unknown type code is still rejected, not silently decoded as Boolean (the zero value).
	if _, err := decodeValue(bytes.NewReader([]byte{9})); err == nil {
		t.Error("decodeValue accepted an unknown type code 9; want an error")
	}
}
