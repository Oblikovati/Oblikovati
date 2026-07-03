// SPDX-License-Identifier: GPL-2.0-only

package meshio

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// Decode parses mesh bytes of the given format into a triangle soup. It is the single
// format switch for import; the per-format decoders own the byte parsing.
//
// Example:
//
//	raw, err := meshio.Decode(types.FormatSTL, data)
func Decode(format types.ExchangeFormat, data []byte) (RawMesh, error) {
	switch format {
	case types.FormatSTL:
		return DecodeSTL(data)
	case types.FormatOBJ:
		return DecodeOBJ(data)
	case types.Format3MF:
		return Decode3MF(data)
	default:
		return RawMesh{}, fmt.Errorf("meshio: unsupported import format %q (want stl|obj|3mf)", format)
	}
}

// ImportBody decodes mesh bytes and welds them into a B-rep body (a solid when
// watertight, a surface body when open), returning any non-fatal warnings. This is the
// import entry point the model layer calls.
//
// Example:
//
//	body, warns, err := meshio.ImportBody(types.FormatSTL, data, "import:stl#0", 0)
func ImportBody(format types.ExchangeFormat, data []byte, feat string, weldTol float64, opts exchange.TranslationOptions) (*topo.Body, []string, error) {
	raw, err := Decode(format, data)
	if err != nil {
		return nil, nil, err
	}
	// One progress tick per decoded mesh, keyed on triangle count, checked BEFORE the expensive
	// weld so a cancel aborts promptly (#1647).
	if err := opts.Report("triangles", len(raw.Tris), len(raw.Tris)); err != nil {
		return nil, nil, err
	}
	fileUnitMM, unitWarns := importFileUnitMM(format, data)
	scaleRaw(&raw, opts.ImportScale(fileUnitMM))
	body, warns, err := SolidOrSurface(raw, feat, weldTol)
	if err != nil {
		return nil, nil, err
	}
	return body, append(unitWarns, warns...), nil
}

// importFileUnitMM is the millimetre size of the file's length unit on import, plus any
// warning about how it was resolved: STL/OBJ are unitless (millimetre convention); 3MF
// declares its unit, read here. An unrecognised (non-empty) 3MF unit spelling falls back to
// millimetre WITH a warning so the user is not silently handed a wrong-scale mesh (#1638); an
// absent unit is the 3MF spec default (millimetre) and warns nothing.
func importFileUnitMM(format types.ExchangeFormat, data []byte) (float64, []string) {
	if format != types.Format3MF {
		return 1, nil // STL/OBJ are unitless — millimetre convention
	}
	unit := read3MFUnit(data)
	if mm, ok := mmPer3MFUnit[unit]; ok {
		return mm, nil
	}
	if unit == "" {
		return 1, nil // absent unit → 3MF spec default is millimetre
	}
	return 1, []string{fmt.Sprintf("3MF declares unknown unit %q; imported as millimetres", unit)}
}

// ExportBody tessellates a body at the given resolution and encodes it in the given
// format. It is the single format switch for export; the per-format encoders own the
// byte emission. The triangle count returned is what was written.
//
// Example:
//
//	data, tris, err := meshio.ExportBody(types.FormatSTL, body, types.ResolutionHigh)
func ExportBody(format types.ExchangeFormat, body *topo.Body, res types.MeshResolution) ([]byte, int, error) {
	q := QualityFor(res)
	switch format {
	case types.FormatSTL:
		data := EncodeBinarySTL(body, q)
		return data, triangleCount(body, q), nil
	case types.FormatOBJ:
		data := EncodeOBJ(body, q)
		return data, triangleCount(body, q), nil
	case types.Format3MF:
		data, err := Encode3MF(body, q)
		return data, triangleCount(body, q), err
	default:
		return nil, 0, fmt.Errorf("meshio: unsupported export format %q (want stl|obj|3mf)", format)
	}
}

// triangleCount returns how many triangles a body tessellates to at quality q (the count
// the encoders write).
func triangleCount(body *topo.Body, q ops.Quality) int {
	mesh, _ := ops.TessellateBody(body, q)
	return mesh.TriangleCount()
}

// ExportBodies tessellates several bodies at the given resolution, merges them into one
// mesh, and encodes it in the given format — the multi-body export path (the mesh formats
// carry a single combined mesh in the first cut). Returns the bytes and triangle count.
//
// Example:
//
//	data, tris, err := meshio.ExportBodies(types.FormatSTL, part.SurfaceBodies().All(), types.ResolutionMedium)
func ExportBodies(format types.ExchangeFormat, bodies []*topo.Body, res types.MeshResolution, opts exchange.TranslationOptions) ([]byte, int, error) {
	// STL and OBJ carry no unit, so they always use the millimetre convention (the
	// universal mesh interchange unit) regardless of the document unit; only 3MF
	// records — and thus honors — the document's unit.
	if format != types.Format3MF {
		opts.FileUnit = "mm"
	}
	q := QualityFor(res)
	merged := mergeTessellations(bodies, q)
	scaleMesh(merged, opts.ExportScale()) // database centimetres → the file unit
	return encodeMesh(format, merged, opts.FileUnit)
}

// encodeMesh encodes an already-tessellated mesh in the given format, returning the bytes
// and triangle count. fileUnit names the 3MF unit attribute (STL/OBJ are unitless).
func encodeMesh(format types.ExchangeFormat, mesh *ops.Mesh, fileUnit string) ([]byte, int, error) {
	switch format {
	case types.FormatSTL:
		return encodeBinarySTLMesh(mesh), mesh.TriangleCount(), nil
	case types.FormatOBJ:
		return encodeOBJMesh(mesh), mesh.TriangleCount(), nil
	case types.Format3MF:
		data, err := encode3MFMesh(mesh, threeMFUnitName(fileUnit))
		return data, mesh.TriangleCount(), err
	default:
		return nil, 0, fmt.Errorf("meshio: unsupported export format %q (want stl|obj|3mf)", format)
	}
}

// mergeTessellations tessellates each body at q and concatenates the meshes (offsetting
// indices) so they export as one combined mesh.
func mergeTessellations(bodies []*topo.Body, q ops.Quality) *ops.Mesh {
	merged := &ops.Mesh{}
	for _, b := range bodies {
		mesh, _ := ops.TessellateBody(b, q)
		base := len(merged.Positions)
		merged.Positions = append(merged.Positions, mesh.Positions...)
		merged.Normals = append(merged.Normals, mesh.Normals...)
		for _, idx := range mesh.Indices {
			merged.Indices = append(merged.Indices, base+idx)
		}
	}
	return merged
}
