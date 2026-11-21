// SPDX-License-Identifier: GPL-2.0-only
package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	stdmath "math"
	"os"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
	"oblikovati.org/persistence"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func requireNear(label string, actual, expected float64) {
	if stdmath.Abs(actual-expected) > 1e-8 {
		panic(fmt.Sprintf("%s %g: expected %g", label, actual, expected))
	}
}
func uniqueFace(part *compdef.PartComponentDefinition, key []byte, area float64) {
	bodies := part.SurfaceBodies().All()
	if len(bodies) != 1 {
		panic(fmt.Sprintf("bodies %d: expected one", len(bodies)))
	}
	matches := 0
	for _, face := range bodies[0].Faces() {
		if !bytes.Equal(face.ReferenceKey(), key) {
			continue
		}
		matches++
		checkFaceArea(face, area)
	}
	if matches != 1 {
		panic(fmt.Sprintf("face key %x: expected one match, got %d", key, matches))
	}
}
func checkFaceArea(face *topo.Face, expected float64) {
	got, exact := topo.NewFaceEvaluator(face).Area()
	if !exact {
		panic("face area: expected exact analytic area")
	}
	requireNear("face area", got, expected)
}
func inspectWork(part *compdef.PartComponentDefinition, key []byte) string {
	rows, err := feature.MarshalWork(part.WorkGeometry())
	must(err)
	if len(rows) != 1 || rows[0].Kind != "plane-offset" || len(rows[0].Refs) != 1 || rows[0].Refs[0] != string(feature.FaceRef(key)) {
		panic(fmt.Sprintf("work definitions %+v: expected one face-offset reference", rows))
	}
	requireNear("saved offset", rows[0].Offset, .5)
	return rows[0].Refs[0]
}
func checkPlane(part *compdef.PartComponentDefinition, x float64) {
	planes := part.WorkPlanes()
	if planes.Count() != 4 {
		panic(fmt.Sprintf("planes %d: expected 4", planes.Count()))
	}
	plane := planes.Item(3)
	if !plane.Health().OK() {
		panic(fmt.Sprintf("plane health %+v: expected healthy", plane.Health()))
	}
	point := plane.Plane().Origin()
	requireNear("plane x", float64(point.X), x)
	requireNear("plane y", float64(point.Y), 0)
	requireNear("plane z", float64(point.Z), 0)
}
func openPart(path string) (*doc.Workspace, *doc.Document, *compdef.PartComponentDefinition) {
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	document, err := ws.Open(path, true)
	must(err)
	return ws, document, document.Content().(*compdef.PartComponentDefinition)
}
func makeFixture(path string, key []byte) string {
	ws, document, part := openPart(path)
	uniqueFace(part, key, 1)
	part.WorkPlanes().AddByPlaneAndOffset(feature.FaceRef(key), func() float64 { return .5 })
	part.Recompute()
	checkPlane(part, 1.5)
	reference := inspectWork(part, key)
	must(ws.Save(document))
	return reference
}
func editSize(part *compdef.PartComponentDefinition) {
	size, ok := part.Parameters().ByName("size")
	if !ok {
		panic("size: expected a parameter")
	}
	must(part.Parameters().SetExpression(size.ID(), "20 mm"))
	part.Features().MarkDirty(part.Features().Item(0))
	part.Recompute()
}
func verifyReplay(path string, key []byte) {
	ws, document, part := openPart(path)
	part.Recompute()
	uniqueFace(part, key, 1)
	checkPlane(part, 1.5)
	inspectWork(part, key)
	editSize(part)
	uniqueFace(part, key, 2)
	checkPlane(part, 2.5)
	must(ws.Save(document))
	_, _, reopened := openPart(path)
	reopened.Recompute()
	uniqueFace(reopened, key, 2)
	checkPlane(reopened, 2.5)
	inspectWork(reopened, key)
}

type observation struct {
	FaceKeyHex         string  `json:"face_key_hex"`
	WorkReference      string  `json:"work_reference"`
	OffsetCM           float64 `json:"offset_cm"`
	InitialPlaneXCM    float64 `json:"initial_plane_x_cm"`
	EditedPlaneXCM     float64 `json:"edited_plane_x_cm"`
	InitialFaceAreaCM2 float64 `json:"initial_face_area_cm2"`
	EditedFaceAreaCM2  float64 `json:"edited_face_area_cm2"`
}

func main() {
	if len(os.Args) != 3 {
		panic("arguments: expected source cube and output file")
	}
	source, err := os.ReadFile(os.Args[1])
	must(err)
	must(os.WriteFile(os.Args[2], source, 0600))
	key, err := hex.DecodeString("03457874727573696f6e313a736964652331")
	must(err)
	ref := makeFixture(os.Args[2], key)
	initial, err := os.ReadFile(os.Args[2])
	must(err)
	defer func() { must(os.WriteFile(os.Args[2], initial, 0600)) }()
	verifyReplay(os.Args[2], key)
	result := observation{hex.EncodeToString(key), ref, .5, 1.5, 2.5, 1, 2}
	encoded, err := json.MarshalIndent(result, "", "  ")
	must(err)
	fmt.Println(string(encoded))
}
