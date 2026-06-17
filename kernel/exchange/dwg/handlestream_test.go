// SPDX-License-Identifier: GPL-2.0-only

package dwg

import "testing"

// TestResolveHandle covers the relative and absolute handle-reference codes (ODA
// dwg_decode_handleref): relatives resolve against the referencing object's own handle.
func TestResolveHandle(t *testing.T) {
	const own = 100
	cases := []struct {
		code  uint8
		value uint64
		want  uint64
	}{
		{0x06, 0, own + 1}, // soft-pointer +1
		{0x08, 0, own - 1}, // -1
		{0x0A, 7, own + 7}, // +offset
		{0x0C, 7, own - 7}, // -offset
		{0x0E, 0, own},     // self
		{0x05, 42, 42},     // hard pointer, absolute
		{0x02, 42, 42},     // absolute
	}
	for _, tc := range cases {
		if got := resolveHandle(tc.code, tc.value, own); got != tc.want {
			t.Errorf("resolveHandle(%#x, %d, %d) = %d, want %d", tc.code, tc.value, own, got, tc.want)
		}
	}
}
