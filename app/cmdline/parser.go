// SPDX-License-Identifier: GPL-2.0-only

package cmdline

import (
	"math"
	"strconv"
	"strings"
)

// The input parser turns the text the user types at a command prompt into typed values
// the engine feeds to a tool: a coordinate, a distance/number, or a keyword option. It
// mirrors AutoCAD's command-line input grammar (M26):
//
//   - absolute cartesian   "10,5"      or 3D "10,5,2"
//   - relative cartesian   "@10,0"     (offset from the previous point)
//   - absolute polar       "10<45"     (distance < angle°, CCW from +X)
//   - relative polar       "@10<45"
//   - a bare distance      "25"        (a value for the active prompt)
//   - a keyword option     "Close"/"C" (matched against the prompt's bracketed options)

// Coord is a point parsed from the command line. Relative is true for "@" input, meaning
// the coordinate is an offset from the previous point rather than a model-space position.
// Polar input is resolved to cartesian here, so the engine only ever sees X/Y/Z.
type Coord struct {
	X, Y, Z  float64
	Relative bool
}

// Fields splits a command line into whitespace-separated tokens. Coordinates use commas,
// not spaces, so "LINE 0,0 10,0" tokenises cleanly to ["LINE", "0,0", "10,0"].
func Fields(line string) []string { return strings.Fields(line) }

// ParseCoord parses one coordinate token (cartesian or polar, absolute or relative),
// returning false when the token is not a coordinate so the caller can try a keyword.
func ParseCoord(tok string) (Coord, bool) {
	tok = strings.TrimSpace(tok)
	rel := strings.HasPrefix(tok, "@")
	if rel {
		tok = tok[1:]
	}
	if tok == "" {
		return Coord{}, false
	}
	if i := strings.IndexByte(tok, '<'); i >= 0 {
		return parsePolar(tok[:i], tok[i+1:], rel)
	}
	return parseCartesian(tok, rel)
}

// parseCartesian parses "x,y" or "x,y,z" into a Coord.
func parseCartesian(tok string, rel bool) (Coord, bool) {
	parts := strings.Split(tok, ",")
	if len(parts) != 2 && len(parts) != 3 {
		return Coord{}, false
	}
	vals := make([]float64, len(parts))
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return Coord{}, false
		}
		vals[i] = v
	}
	c := Coord{X: vals[0], Y: vals[1], Relative: rel}
	if len(vals) == 3 {
		c.Z = vals[2]
	}
	return c, true
}

// parsePolar resolves "distance<angle°" (CCW from +X) into a cartesian Coord.
func parsePolar(distStr, angStr string, rel bool) (Coord, bool) {
	dist, err := strconv.ParseFloat(strings.TrimSpace(distStr), 64)
	if err != nil {
		return Coord{}, false
	}
	ang, err := strconv.ParseFloat(strings.TrimSpace(angStr), 64)
	if err != nil {
		return Coord{}, false
	}
	rad := ang * math.Pi / 180
	return Coord{X: dist * math.Cos(rad), Y: dist * math.Sin(rad), Relative: rel}, true
}

// ParseDistance parses a bare numeric value (a length, radius, depth, or angle the active
// prompt asks for), returning false when the token is not a number.
func ParseDistance(tok string) (float64, bool) {
	v, err := strconv.ParseFloat(strings.TrimSpace(tok), 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// MatchKeyword resolves a typed token against a prompt's bracketed options (e.g. [Close
// Undo]), AutoCAD-style: a case-insensitive exact match wins, else a single unambiguous
// case-insensitive prefix. An empty token or an ambiguous prefix returns false.
func MatchKeyword(tok string, options []string) (string, bool) {
	tok = strings.TrimSpace(tok)
	if tok == "" {
		return "", false
	}
	for _, o := range options {
		if strings.EqualFold(o, tok) {
			return o, true
		}
	}
	var hit string
	var n int
	for _, o := range options {
		if strings.HasPrefix(strings.ToLower(o), strings.ToLower(tok)) {
			hit, n = o, n+1
		}
	}
	if n == 1 {
		return hit, true
	}
	return "", false
}
