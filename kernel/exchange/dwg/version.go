// SPDX-License-Identifier: GPL-2.0-only

package dwg

import "fmt"

// Version identifies a DWG format generation, selected from the six magic bytes at
// the very start of the file. The structural layout of the rest of the file
// branches on this: R13–R2000 use flat sentinel-bounded sections, while R2004 and
// later use a paged, compressed (and partly Reed-Solomon-protected) container.
type Version int

const (
	VersionUnknown Version = iota
	R13                    // AC1012 — AutoCAD R13
	R14                    // AC1014 — AutoCAD R14
	R2000                  // AC1015 — AutoCAD 2000/2002 (flat sections)
	R2004                  // AC1018 — AutoCAD 2004/2006 (paged + LZ77)
	R2007                  // AC1021 — AutoCAD 2007/2009 (unicode, RS pages)
	R2010                  // AC1024 — AutoCAD 2010/2012
	R2013                  // AC1027 — AutoCAD 2013/2017
	R2018                  // AC1032 — AutoCAD 2018+ (modern paged container)
)

// magicToVersion maps the 6-byte ASCII signature to its [Version].
var magicToVersion = map[string]Version{
	"AC1012": R13,
	"AC1014": R14,
	"AC1015": R2000,
	"AC1018": R2004,
	"AC1021": R2007,
	"AC1024": R2010,
	"AC1027": R2013,
	"AC1032": R2018,
}

// versionMagic is the inverse of magicToVersion, for [Version.Magic]/encoding.
var versionMagic = func() map[Version]string {
	m := make(map[Version]string, len(magicToVersion))
	for k, v := range magicToVersion {
		m[v] = k
	}
	return m
}()

// Magic returns the 6-byte ASCII signature for v, or "" for [VersionUnknown].
func (v Version) Magic() string { return versionMagic[v] }

// Paged reports whether the version uses the R2004+ paged/compressed container
// (as opposed to the flat R13–R2000 section layout). This is the single branch
// point most of the container code keys on.
func (v Version) Paged() bool { return v >= R2004 }

// String renders the AutoCAD release name, e.g. "R2018 (AC1032)".
func (v Version) String() string {
	if m := v.Magic(); m != "" {
		return fmt.Sprintf("%s (%s)", v.name(), m)
	}
	return "unknown"
}

func (v Version) name() string {
	names := map[Version]string{
		R13: "R13", R14: "R14", R2000: "R2000", R2004: "R2004",
		R2007: "R2007", R2010: "R2010", R2013: "R2013", R2018: "R2018",
	}
	return names[v]
}

// DetectVersion reads the leading magic bytes and returns the format generation.
// It errors (rather than returning VersionUnknown) when the data is too short or
// the signature is unrecognised, so callers get the offending bytes in the message.
//
// Example:
//
//	v, err := dwg.DetectVersion(data) // v == dwg.R2018 for an AC1032 file
func DetectVersion(data []byte) (Version, error) {
	if len(data) < 6 {
		return VersionUnknown, fmt.Errorf("dwg: file too short for magic: have %d bytes, need >= 6", len(data))
	}
	magic := string(data[:6])
	v, ok := magicToVersion[magic]
	if !ok {
		return VersionUnknown, fmt.Errorf("dwg: unrecognised signature %q (supported: AC1012/AC1014/AC1015/AC1018/AC1021/AC1024/AC1027/AC1032)", magic)
	}
	return v, nil
}
