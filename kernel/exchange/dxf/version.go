// SPDX-License-Identifier: GPL-2.0-only

package dxf

// Version is a DXF format generation. The codec targets two on export, selectable by the
// user; the geometry group codes are identical across them — only the $ACADVER string and a
// little section scaffolding differ.
type Version int

const (
	R2000 Version = iota // AC1015
	R2018                // AC1032
)

// acadVer maps a Version to its $ACADVER header string.
var acadVer = map[Version]string{
	R2000: "AC1015",
	R2018: "AC1032",
}

// ACADVer returns the $ACADVER string the encoder writes for this version.
func (v Version) ACADVer() string {
	if s, ok := acadVer[v]; ok {
		return s
	}
	return acadVer[R2000]
}
