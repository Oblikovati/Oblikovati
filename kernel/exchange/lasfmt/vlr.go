// SPDX-License-Identifier: GPL-2.0-only

package lasfmt

import (
	"bytes"
	"encoding/binary"
	"math"
	"strconv"
	"strings"
)

// LAS carries its coordinate reference system in Variable Length Records (VLRs) under the
// "LASF_Projection" user id: either an OGC WKT string (record id 2112, the LAS 1.4 default) or a
// GeoTIFF GeoKey directory (record id 34735, with its double params in 34736). Both spell out the
// horizontal linear unit and its size in metres, which a point-cloud import needs to place the scan
// at true scale — a survey in US survey feet is otherwise read as metres (~3.28x too large) and a
// millimetre-declared scan 1000x too large (Oblikovati/Oblikovati#1789). This file walks the VLRs
// and extracts that metres-per-unit factor.

const (
	vlrHeaderSize     = 54 // reserved(2) + userID(16) + recordID(2) + recordLen(2) + description(32)
	crsUserID         = "LASF_Projection"
	wktRecordID       = 2112  // OGC Coordinate System WKT
	geoKeyDirRecordID = 34735 // GeoKeyDirectoryTag
	geoDoubleRecordID = 34736 // GeoDoubleParamsTag

	projLinearUnitsGeoKey = 3076  // EPSG code of the projected CRS's linear unit
	projLinearSizeGeoKey  = 3077  // user-defined linear unit size in metres (when 3076 == user-defined)
	geoKeyUserDefined     = 32767 // "the value is user-defined; read the size key"
)

// coordinateUnitMetres returns the horizontal linear unit's size in metres declared by the file's
// CRS VLRs (e.g. 1.0 for metre, 0.3048 for the international foot, 0.30480060960121924 for the US
// survey foot), and whether one was found. WKT is preferred over the GeoTIFF GeoKeys when both are
// present (it is the LAS 1.4 canonical form and carries the exact factor); a geographic-only CRS
// (degrees, no linear XY) yields ok=false.
func coordinateUnitMetres(data []byte, h lasHeader) (float64, bool) {
	wkt, geoDir, geoDoubles := crsVLRs(data, h)
	if f, ok := wktLinearUnitMetres(wkt); ok {
		return f, true
	}
	return geoKeyLinearUnitMetres(geoDir, geoDoubles)
}

// crsVLRs walks the VLR block after the public header and returns the raw payloads of the CRS
// records: the WKT string and the GeoKey directory plus its double-params array. Missing records
// come back empty. Malformed lengths stop the walk rather than reading past the file.
func crsVLRs(data []byte, h lasHeader) (wkt string, geoDir, geoDoubles []byte) {
	pos := int(h.headerSize)
	for i := uint32(0); i < h.vlrCount; i++ {
		userID, recordID, payload, next, ok := readVLR(data, pos)
		if !ok {
			return
		}
		pos = next
		if userID != crsUserID {
			continue
		}
		switch recordID {
		case wktRecordID:
			wkt = cString(payload)
		case geoKeyDirRecordID:
			geoDir = payload
		case geoDoubleRecordID:
			geoDoubles = payload
		}
	}
	return
}

// readVLR parses the VLR at pos, returning its user id, record id, payload, and the offset of the
// next VLR. ok is false when the record header or its declared length runs past the file.
func readVLR(data []byte, pos int) (userID string, recordID uint16, payload []byte, next int, ok bool) {
	if pos+vlrHeaderSize > len(data) {
		return "", 0, nil, 0, false
	}
	recordLen := int(binary.LittleEndian.Uint16(data[pos+20:]))
	body := pos + vlrHeaderSize
	if body+recordLen > len(data) {
		return "", 0, nil, 0, false
	}
	return cString(data[pos+2 : pos+18]), binary.LittleEndian.Uint16(data[pos+18:]), data[body : body+recordLen], body + recordLen, true
}

// wktLinearUnitMetres extracts the linear unit's metre conversion factor from an OGC WKT string. The
// factor is the second argument of the linear unit keyword, so no unit-name table is needed. Only a
// projected CRS has a linear XY unit; a geographic CRS (degrees) yields ok=false. WKT2 tags the
// linear unit LENGTHUNIT (angular is ANGLEUNIT); WKT1 uses UNIT, whose last occurrence in a PROJCS
// is the linear one (the earlier UNIT is the base geographic degree).
func wktLinearUnitMetres(wkt string) (float64, bool) {
	if wkt == "" {
		return 0, false
	}
	up := strings.ToUpper(wkt)
	if !strings.Contains(up, "PROJCRS") && !strings.Contains(up, "PROJCS") {
		return 0, false // geographic (degrees) — not a linear scan unit
	}
	if f, ok := lastUnitFactor(up, "LENGTHUNIT"); ok { // WKT2
		return f, true
	}
	return lastUnitFactor(up, "UNIT") // WKT1
}

// lastUnitFactor reads the metre-conversion factor from the last `KEYWORD["name",factor…]` in wkt.
// It takes the LAST occurrence because a projected CRS nests the base geographic unit before its own
// coordinate unit.
func lastUnitFactor(wkt, keyword string) (float64, bool) {
	idx := strings.LastIndex(wkt, keyword+"[")
	if idx < 0 {
		return 0, false
	}
	return unitFactorAfterName(wkt[idx+len(keyword)+1:])
}

// unitFactorAfterName parses `"unit name",factor…` and returns the factor, skipping the quoted name
// so a comma or bracket inside it cannot confuse the parse.
func unitFactorAfterName(s string) (float64, bool) {
	open := strings.IndexByte(s, '"')
	if open < 0 {
		return 0, false
	}
	end := strings.IndexByte(s[open+1:], '"')
	if end < 0 {
		return 0, false
	}
	s = s[open+1+end+1:]
	comma := strings.IndexByte(s, ',')
	if comma < 0 {
		return 0, false
	}
	stop := strings.IndexAny(s[comma+1:], ",]")
	if stop < 0 {
		stop = len(s) - comma - 1
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(s[comma+1:comma+1+stop]), 64)
	if err != nil || f <= 0 {
		return 0, false
	}
	return f, true
}

// geoKeyLinearUnitMetres reads ProjLinearUnitsGeoKey (3076) from a GeoTIFF GeoKey directory. An
// inline EPSG code maps through epsgLinearMetres; a user-defined unit (32767) reads its metre size
// from ProjLinearUnitSizeGeoKey (3077) in the GeoDoubleParams array.
func geoKeyLinearUnitMetres(dir, doubles []byte) (float64, bool) {
	unit, sizeIndex, hasSize, ok := scanLinearUnitKeys(dir)
	if !ok {
		return 0, false
	}
	if unit == geoKeyUserDefined && hasSize {
		return geoDoubleAt(doubles, sizeIndex)
	}
	if unit == 0 {
		return 0, false
	}
	return epsgLinearMetres(unit)
}

// scanLinearUnitKeys reads ProjLinearUnitsGeoKey (the unit code) and ProjLinearUnitSizeGeoKey (its
// GeoDoubleParams index, for a user-defined unit) from a GeoKey directory of [4-uint16 header, then
// 4 uint16 per key]. ok is false on a malformed directory or a units key stored anywhere but inline.
func scanLinearUnitKeys(dir []byte) (unit, sizeIndex int, hasSize, ok bool) {
	if len(dir) < 8 {
		return 0, 0, false, false
	}
	u16 := func(i int) uint16 { return binary.LittleEndian.Uint16(dir[i*2:]) }
	numKeys := int(u16(3))
	for k := range numKeys {
		base := 4 + k*4
		if (base+4)*2 > len(dir) {
			break
		}
		keyID, tagLoc, valOff := u16(base), u16(base+1), u16(base+3)
		switch keyID {
		case projLinearUnitsGeoKey:
			if tagLoc != 0 { // linear units are a SHORT stored inline (tagLoc 0); anything else is malformed
				return 0, 0, false, false
			}
			unit = int(valOff)
		case projLinearSizeGeoKey:
			if tagLoc == geoDoubleRecordID { // the size lives in the GeoDoubleParams VLR (34736)
				sizeIndex, hasSize = int(valOff), true
			}
		}
	}
	return unit, sizeIndex, hasSize, true
}

// epsgLinearMetres maps the common EPSG linear-unit codes to their size in metres. Unknown codes
// yield ok=false so the caller falls back to the quantisation heuristic rather than guessing.
func epsgLinearMetres(code int) (float64, bool) {
	switch code {
	case 9001:
		return 1.0, true // metre
	case 9002:
		return 0.3048, true // international foot
	case 9003:
		return 0.30480060960121924, true // US survey foot (1200/3937)
	default:
		return 0, false
	}
}

// geoDoubleAt returns the float64 at index i of the GeoDoubleParams array (8 bytes each).
func geoDoubleAt(doubles []byte, i int) (float64, bool) {
	if i < 0 || (i+1)*8 > len(doubles) {
		return 0, false
	}
	v := math.Float64frombits(binary.LittleEndian.Uint64(doubles[i*8:]))
	if v <= 0 {
		return 0, false
	}
	return v, true
}

// cString trims a fixed-width byte field at its first NUL (LAS pads userID/description with NULs).
func cString(b []byte) string {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return strings.TrimRight(string(b), " ")
}
