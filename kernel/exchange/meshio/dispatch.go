// SPDX-License-Identifier: GPL-2.0-only

package meshio

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
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
// The legacy mesh formats (STL/OBJ/3MF) tessellate ONCE via [TessellateBodies] and
// feed the same mesh to the encoder and the count — the per-body encoders
// (EncodeBinarySTL/EncodeOBJ/Encode3MF) tessellate internally, so encoding and
// counting separately would tessellate twice (CHG4-1). The file unit is the legacy
// single-body default: 3MF declares the 3MF document unit default millimetre
// (Encode3MF); STL/OBJ are unitless.
//
// glTF delegates to the per-body path [ExportBodiesGLTF] with a single-body slice and
// zero-value options (the glTF path defaults the unit itself — see CHG3-2). Its
// warnings (a skipped empty body) have nowhere to go in this 3-value signature, so
// they are dropped here; callers that need them use [ExportBodiesGLTF] directly. A
// single empty body errors with "no exportable bodies", which is correct for a
// single empty body (CHG3-1).
//
// Example:
//
//	data, tris, err := meshio.ExportBody(types.FormatSTL, body, types.ResolutionHigh)
func ExportBody(format types.ExchangeFormat, body *topo.Body, res types.MeshResolution) ([]byte, int, error) {
	q := QualityFor(res)
	switch format {
	case types.FormatGLTF:
		data, tris, _, err := ExportBodiesGLTF([]*topo.Body{body}, res, exchange.TranslationOptions{})
		return data, tris, err
	case types.FormatSTL, types.FormatOBJ, types.Format3MF:
		records, err := TessellateBodies([]*topo.Body{body}, q)
		if err != nil {
			return nil, 0, err
		}
		return encodeMesh(format, records[0].Mesh, "millimeter")
	default:
		return nil, 0, fmt.Errorf("meshio: unsupported export format %q (want stl|obj|3mf|gltf)", format)
	}
}

// ExportBodies tessellates several bodies at the given resolution, merges them into one
// mesh, and encodes it in the given format — the multi-body export path (the mesh formats
// carry a single combined mesh in the first cut). Returns the bytes and triangle count.
//
// Example:
//
//	data, tris, err := meshio.ExportBodies(types.FormatSTL, part.SurfaceBodies().All(), types.ResolutionMedium)
func ExportBodies(format types.ExchangeFormat, bodies []*topo.Body, res types.MeshResolution, opts exchange.TranslationOptions) ([]byte, int, error) {
	// glTF is a per-body format (one mesh per body, one primitive per mesh) and
	// has its own entry point — the merged path below would destroy the per-body
	// identity glTF's node design needs (R4-1).
	if format == types.FormatGLTF {
		return nil, 0, fmt.Errorf("meshio: gltf export requires the per-body path (ExportBodiesGLTF)")
	}
	// The file unit is forced per format on a LOCAL copy so the caller's options
	// are never mutated (R3-3): STL/OBJ are unitless — millimetre convention;
	// only 3MF records — and thus honors — the document's unit.
	local := opts
	if format != types.Format3MF {
		local.FileUnit = "mm"
	}
	q := QualityFor(res)
	merged, err := mergeTessellations(bodies, q)
	if err != nil {
		return nil, 0, err
	}
	scaleMesh(merged, local.ExportScale()) // database centimetres → the file unit
	return encodeMesh(format, merged, local.FileUnit)
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
		return nil, 0, fmt.Errorf("meshio: unsupported export format %q (want stl|obj|3mf|gltf)", format)
	}
}

// BodyMesh is one body's tessellation record: the stable body identifier, the
// derived stable label, and the tessellated mesh. It is the per-body unit the
// glTF exporter consumes (R2-7/R4-1).
//
// Name is a DERIVED stable label ("Body<id>"), NOT a source display name: the
// kernel's topo.Body carries no display-name surface (only ID/Kind/Lineage/
// ReferenceKey), so the exporter cannot know a body's user-facing name.
// Display-name plumbing from the model/occurrence layer into the kernel is a
// recorded follow-up (change-review CHG-5); until it lands, exported names are
// the collision-safe Body<id> labels.
type BodyMesh struct {
	ID   string
	Name string
	Mesh *ops.Mesh
}

// TessellateBodies tessellates each body at q, in input slice order (kernel
// B-rep order, stable — R2-10). It is the SINGLE ownership point of
// tessellation: the glTF branch consumes its records directly and never
// merges; mergeTessellations is a pure merger over the same records, so every
// mesh format tessellates exactly once (R4-1). A body that fails to tessellate
// (or yields a nil mesh) is a typed error naming the body id — never a silent
// skip (change-review CHG-3).
//
// Example:
//
//	meshes, err := meshio.TessellateBodies(bodies, meshio.QualityFor(types.ResolutionHigh))
func TessellateBodies(bodies []*topo.Body, q ops.Quality) ([]BodyMesh, error) {
	out := make([]BodyMesh, 0, len(bodies))
	for i, b := range bodies {
		if b == nil {
			return nil, fmt.Errorf("tessellate: body at index %d is nil", i)
		}
		mesh, _ := tessellate.TessellateBody(b, q)
		if mesh == nil {
			return nil, fmt.Errorf("tessellate body %d: nil mesh", b.ID())
		}
		out = append(out, BodyMesh{ID: fmt.Sprintf("%d", b.ID()), Name: fmt.Sprintf("Body%d", b.ID()), Mesh: mesh})
	}
	return out, nil
}

// mergeTessellations is a PURE merger: it tessellates the bodies once via
// [TessellateBodies] and concatenates the meshes (offsetting indices) so they
// export as one combined mesh — the legacy STL/OBJ/3MF shape, byte-for-byte
// unchanged (R4-1).
func mergeTessellations(bodies []*topo.Body, q ops.Quality) (*ops.Mesh, error) {
	merged := &ops.Mesh{}
	records, err := TessellateBodies(bodies, q)
	if err != nil {
		return nil, err
	}
	for _, bm := range records {
		mesh := bm.Mesh
		base := len(merged.Positions)
		merged.Positions = append(merged.Positions, mesh.Positions...)
		merged.Normals = append(merged.Normals, mesh.Normals...)
		for _, idx := range mesh.Indices {
			merged.Indices = append(merged.Indices, base+idx)
		}
	}
	return merged, nil
}
