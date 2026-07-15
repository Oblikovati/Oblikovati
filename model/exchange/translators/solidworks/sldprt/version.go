// SPDX-License-Identifier: GPL-2.0-only

package sldprt

import "strings"

// versionOf reads the internal build version from the "_MO_VERSION_NNNN" storage that
// SolidWorks stamps into every document (e.g. "_MO_VERSION_3400/Biography" -> 3400).
// The number is the modeler-object schema version, which pins the byte layout of the
// decoded streams; 0 if absent.
func versionOf(streams []string) int {
	const prefix = "_MO_VERSION_"
	for _, s := range streams {
		i := strings.Index(s, prefix)
		if i < 0 {
			continue
		}
		rest := s[i+len(prefix):]
		if slash := strings.IndexByte(rest, '/'); slash >= 0 {
			rest = rest[:slash]
		}
		if n, ok := atoiPositive(rest); ok {
			return n
		}
	}
	return 0
}

// atoiPositive parses a run of ASCII digits into a non-negative int, rejecting anything
// else (avoids strconv for a hot, fixed-shape field).
func atoiPositive(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
		n = n*10 + int(s[i]-'0')
	}
	return n, true
}
