// SPDX-License-Identifier: GPL-2.0-only

package dxf

import (
	"oblikovati.org/kernel/exchange/drawing"
)

// Decode parses an ASCII DXF file and returns its model-space curve geometry plus the
// $INSUNITS code. Like the DWG decoder, per-entity failures are collected as warnings
// rather than aborting the whole drawing, so one malformed record never sinks the import.
// Unknown sections and entity types are skipped.
//
// Example:
//
//	dr, warns, err := dxf.Decode(bytes)
//	for _, e := range dr.Entities { /* convert to sketch geometry */ }
func Decode(data []byte) (*drawing.Drawing, []string, error) {
	pairs, err := scanPairs(data)
	if err != nil {
		return nil, nil, err
	}
	dr := &drawing.Drawing{Units: drawing.INSUnitless}
	var warns []string
	sections := splitSections(pairs)
	if hdr, ok := sections["HEADER"]; ok {
		dr.Units = headerInsunits(hdr)
	}
	if ents, ok := sections["ENTITIES"]; ok {
		dr.Entities, warns = decodeEntities(ents)
	}
	return dr, warns, nil
}

// splitSections groups the pair stream by DXF section, returning each section's inner pairs
// (the SECTION/2/name preamble and the ENDSEC terminator are stripped). A later section of
// the same name overwrites an earlier one (DXF has at most one of each).
func splitSections(pairs []pair) map[string][]pair {
	out := map[string][]pair{}
	for i := 0; i < len(pairs); i++ {
		if pairs[i].code != 0 || pairs[i].value != "SECTION" {
			continue
		}
		// The pair after SECTION is code 2 with the section name.
		if i+1 >= len(pairs) || pairs[i+1].code != 2 {
			continue
		}
		name := pairs[i+1].text()
		body, next := sectionBody(pairs, i+2)
		out[name] = body
		i = next
	}
	return out
}

// sectionBody returns the pairs from start up to (not including) the ENDSEC marker, and the
// index of that marker so the caller can resume after it. A section missing its ENDSEC runs
// to end of file.
func sectionBody(pairs []pair, start int) (body []pair, endsec int) {
	for j := start; j < len(pairs); j++ {
		if pairs[j].code == 0 && pairs[j].value == "ENDSEC" {
			return pairs[start:j], j
		}
	}
	return pairs[start:], len(pairs)
}

// headerInsunits scans the HEADER section for the $INSUNITS variable (code 9 name followed
// by its code-70 value). Absent or malformed, it returns unitless (0).
func headerInsunits(hdr []pair) int {
	for i := 0; i+1 < len(hdr); i++ {
		if hdr[i].code == 9 && hdr[i].text() == "$INSUNITS" {
			if v, err := hdr[i+1].integer(); err == nil {
				return v
			}
			return drawing.INSUnitless
		}
	}
	return drawing.INSUnitless
}
