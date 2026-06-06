// SPDX-License-Identifier: GPL-2.0-only

package meshio

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
	"oblikovati/math"
)

// DecodeOBJ decodes a Wavefront OBJ into a triangle soup, reading only `v` (vertex) and
// `f` (face) records; texture/normal/material lines (vt/vn/usemtl/mtllib) are ignored in
// this first cut. Faces with >3 corners are triangulated as a fan. Face indices are
// 1-based; negative indices count back from the end (the OBJ relative form). It errors on
// a malformed coordinate or an out-of-range index, naming the offending value.
//
// Example:
//
//	raw, err := meshio.DecodeOBJ(data)
func DecodeOBJ(data []byte) (RawMesh, error) {
	sc := bufio.NewScanner(bytes.NewReader(data))
	var verts []math.Point3
	var m RawMesh
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 {
			continue
		}
		if err := decodeOBJLine(fields, &verts, &m); err != nil {
			return RawMesh{}, err
		}
	}
	return m, sc.Err()
}

// decodeOBJLine dispatches one OBJ record: a `v` grows verts, an `f` adds a face to m,
// anything else is ignored.
func decodeOBJLine(fields []string, verts *[]math.Point3, m *RawMesh) error {
	switch fields[0] {
	case "v":
		p, err := parseOBJVertex(fields)
		if err != nil {
			return err
		}
		*verts = append(*verts, p)
	case "f":
		return parseOBJFace(fields, *verts, m)
	}
	return nil
}

// parseOBJVertex reads the three coordinates of a `v x y z` record.
func parseOBJVertex(fields []string) (math.Point3, error) {
	if len(fields) < 4 {
		return math.Point3{}, &decodeError{format: "OBJ", what: "vertex needs 3 coordinates, got", value: strings.Join(fields, " ")}
	}
	var c [3]float64
	for i := 0; i < 3; i++ {
		v, err := strconv.ParseFloat(fields[i+1], 64)
		if err != nil {
			return math.Point3{}, &decodeError{format: "OBJ", what: "bad coordinate", value: fields[i+1]}
		}
		c[i] = v
	}
	return math.P3(c[0], c[1], c[2]), nil
}

// parseOBJFace resolves a face's corner indices (fan-triangulated) into m, referencing
// the already-collected verts.
func parseOBJFace(fields []string, verts []math.Point3, m *RawMesh) error {
	idx := make([]int, 0, len(fields)-1)
	for _, tok := range fields[1:] {
		i, err := resolveOBJIndex(tok, len(verts))
		if err != nil {
			return err
		}
		idx = append(idx, i)
	}
	for i := 2; i < len(idx); i++ {
		m.AddTriangle(verts[idx[0]], verts[idx[i-1]], verts[idx[i]])
	}
	return nil
}

// resolveOBJIndex parses a face corner token ("v", "v/vt", "v/vt/vn", "v//vn") to a
// 0-based vertex index, handling the 1-based and negative-relative OBJ conventions.
func resolveOBJIndex(tok string, n int) (int, error) {
	first := strings.SplitN(tok, "/", 2)[0]
	v, err := strconv.Atoi(first)
	if err != nil {
		return 0, &decodeError{format: "OBJ", what: "bad face index", value: tok}
	}
	if v < 0 {
		v = n + v // -1 ⇒ last vertex
	} else {
		v-- // 1-based ⇒ 0-based
	}
	if v < 0 || v >= n {
		return 0, &decodeError{format: "OBJ", what: "face index out of range", value: tok}
	}
	return v, nil
}

// EncodeOBJ tessellates body at quality q and writes a Wavefront OBJ (v + f records,
// 1-based indices). The resolution knob applies through q.
//
// Example:
//
//	data := meshio.EncodeOBJ(body, meshio.QualityFor(types.ResolutionMedium))
func EncodeOBJ(body *topo.Body, q ops.Quality) []byte {
	mesh, _ := ops.TessellateBody(body, q)
	return encodeOBJMesh(mesh)
}

// encodeOBJMesh writes an already-tessellated mesh as a Wavefront OBJ.
func encodeOBJMesh(mesh *ops.Mesh) []byte {
	var buf bytes.Buffer
	buf.WriteString("# Oblikovati mesh export\n")
	for _, p := range mesh.Positions {
		fmt.Fprintf(&buf, "v %g %g %g\n", p.X, p.Y, p.Z)
	}
	for t := 0; t+2 < len(mesh.Indices); t += 3 {
		fmt.Fprintf(&buf, "f %d %d %d\n", mesh.Indices[t]+1, mesh.Indices[t+1]+1, mesh.Indices[t+2]+1)
	}
	return buf.Bytes()
}
