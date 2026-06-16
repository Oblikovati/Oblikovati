// SPDX-License-Identifier: GPL-2.0-only

package dwg

// ParseFileHeader detects the DWG generation from the magic bytes and decodes the
// top-level container layout into a [FileHeader]. R13–R2000 use the flat locator
// table; R2004+ use the paged/compressed container.
//
// Example:
//
//	h, err := dwg.ParseFileHeader(data)
//	objs, _ := h.SectionBytes(data, 2) // R2000 object-map bytes
func ParseFileHeader(data []byte) (*FileHeader, error) {
	version, err := DetectVersion(data)
	if err != nil {
		return nil, err
	}
	if version.Paged() {
		return parseHeaderR2004(data, version)
	}
	return parseHeaderR2000(data, version)
}
