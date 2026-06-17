// SPDX-License-Identifier: GPL-2.0-only

package meshio

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// modelPartPath is the OPC part the 3MF spec mandates for the 3D model XML.
const modelPartPath = "3D/3dmodel.model"

// threeMFModel is the subset of the 3MF <model> XML we read/write: one object whose mesh
// carries vertices and triangles. Materials, components and assemblies are out of scope
// for the first cut (a single faceted object covers import→solid and export round-trip).
type threeMFModel struct {
	XMLName   xml.Name        `xml:"model"`
	Unit      string          `xml:"unit,attr"`
	Resources threeMFResource `xml:"resources"`
}

type threeMFResource struct {
	Objects []threeMFObject `xml:"object"`
}

type threeMFObject struct {
	Mesh threeMFMesh `xml:"mesh"`
}

type threeMFMesh struct {
	Vertices  []threeMFVertex   `xml:"vertices>vertex"`
	Triangles []threeMFTriangle `xml:"triangles>triangle"`
}

type threeMFVertex struct {
	X float64 `xml:"x,attr"`
	Y float64 `xml:"y,attr"`
	Z float64 `xml:"z,attr"`
}

type threeMFTriangle struct {
	V1 int `xml:"v1,attr"`
	V2 int `xml:"v2,attr"`
	V3 int `xml:"v3,attr"`
}

// Decode3MF decodes a 3MF file (a ZIP container) into a triangle soup by reading the
// 3D/3dmodel.model XML part and flattening every object's mesh. It errors on a missing
// model part, malformed ZIP/XML, or an out-of-range triangle index, naming the cause.
//
// Example:
//
//	raw, err := meshio.Decode3MF(data)
func Decode3MF(data []byte) (RawMesh, error) {
	xmlBytes, err := readModelPart(data)
	if err != nil {
		return RawMesh{}, err
	}
	var model threeMFModel
	if err := xml.Unmarshal(xmlBytes, &model); err != nil {
		return RawMesh{}, &decodeError{format: "3MF", what: "malformed model XML", value: err.Error()}
	}
	return soupFromModel(model)
}

// read3MFUnit returns the <model unit="…"> spelling, or "" when it cannot be read
// (the caller then falls back to millimetres). Used to scale a 3MF import into the
// database unit.
func read3MFUnit(data []byte) string {
	xmlBytes, err := readModelPart(data)
	if err != nil {
		return ""
	}
	var model threeMFModel
	if xml.Unmarshal(xmlBytes, &model) != nil {
		return ""
	}
	return model.Unit
}

// readModelPart opens the ZIP and returns the bytes of the 3D/3dmodel.model part.
func readModelPart(data []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, &decodeError{format: "3MF", what: "not a ZIP container", value: err.Error()}
	}
	for _, f := range zr.File {
		if f.Name != modelPartPath {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, &decodeError{format: "3MF", what: "cannot open model part", value: f.Name}
		}
		defer rc.Close()
		return io.ReadAll(rc)
	}
	return nil, &decodeError{format: "3MF", what: "missing model part", value: modelPartPath}
}

// soupFromModel flattens every object's mesh into one triangle soup, validating indices.
func soupFromModel(model threeMFModel) (RawMesh, error) {
	var m RawMesh
	for oi, obj := range model.Resources.Objects {
		verts := make([]math.Point3, len(obj.Mesh.Vertices))
		for i, v := range obj.Mesh.Vertices {
			verts[i] = math.P3(v.X, v.Y, v.Z)
		}
		for _, t := range obj.Mesh.Triangles {
			if err := addTriangleByIndex(&m, verts, t, oi); err != nil {
				return RawMesh{}, err
			}
		}
	}
	if len(m.Tris) == 0 {
		return RawMesh{}, &decodeError{format: "3MF", what: "no triangles in object count", value: fmt.Sprint(len(model.Resources.Objects))}
	}
	return m, nil
}

// addTriangleByIndex appends a triangle resolved against verts, erroring on a bad index.
func addTriangleByIndex(m *RawMesh, verts []math.Point3, t threeMFTriangle, obj int) error {
	for _, i := range []int{t.V1, t.V2, t.V3} {
		if i < 0 || i >= len(verts) {
			return &decodeError{format: "3MF", what: fmt.Sprintf("object %d triangle index out of range", obj), value: fmt.Sprint(i)}
		}
	}
	m.AddTriangle(verts[t.V1], verts[t.V2], verts[t.V3])
	return nil
}

// Encode3MF tessellates body at quality q and writes a minimal valid 3MF: a ZIP holding
// the OPC content-types/relationships parts and the 3D model XML with one object. The
// resolution knob applies through q.
//
// Example:
//
//	data, err := meshio.Encode3MF(body, meshio.QualityFor(types.ResolutionHigh))
func Encode3MF(body *topo.Body, q ops.Quality) ([]byte, error) {
	mesh, _ := ops.TessellateBody(body, q)
	return encode3MFMesh(mesh, "millimeter")
}

// encode3MFMesh writes an already-tessellated mesh as a 3MF ZIP container declaring
// the given 3MF unit spelling (e.g. "millimeter", "inch").
func encode3MFMesh(mesh *ops.Mesh, unit string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	parts := map[string][]byte{
		"[Content_Types].xml": []byte(contentTypesXML),
		"_rels/.rels":         []byte(relsXML),
		modelPartPath:         modelXML(mesh, unit),
	}
	for name, content := range parts {
		if err := writeZipPart(zw, name, content); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("3MF: close zip: %w", err)
	}
	return buf.Bytes(), nil
}

// writeZipPart writes one named part into the zip.
func writeZipPart(zw *zip.Writer, name string, content []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return fmt.Errorf("3MF: create part %q: %w", name, err)
	}
	_, err = w.Write(content)
	return err
}
