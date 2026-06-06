// SPDX-License-Identifier: GPL-2.0-only

package meshio

import (
	"fmt"

	"oblikovati/api/types"
	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
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
func ImportBody(format types.ExchangeFormat, data []byte, feat string, weldTol float64) (*topo.Body, []string, error) {
	raw, err := Decode(format, data)
	if err != nil {
		return nil, nil, err
	}
	return SolidOrSurface(raw, feat, weldTol)
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
func ExportBodies(format types.ExchangeFormat, bodies []*topo.Body, res types.MeshResolution) ([]byte, int, error) {
	q := QualityFor(res)
	merged := mergeTessellations(bodies, q)
	return encodeMesh(format, merged)
}

// encodeMesh encodes an already-tessellated mesh in the given format, returning the bytes
// and triangle count.
func encodeMesh(format types.ExchangeFormat, mesh *ops.Mesh) ([]byte, int, error) {
	switch format {
	case types.FormatSTL:
		return encodeBinarySTLMesh(mesh), mesh.TriangleCount(), nil
	case types.FormatOBJ:
		return encodeOBJMesh(mesh), mesh.TriangleCount(), nil
	case types.Format3MF:
		data, err := encode3MFMesh(mesh)
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
