// SPDX-License-Identifier: GPL-2.0-only
// Print the legacy analytic face frames used by the Rust oracle test.
package main

import (
	"encoding/hex"
	"fmt"
	"os"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/persistence"
)

func printFace(face *topo.Face) {
	plane, ok := face.Geometry().(geom.Plane)
	if !ok {
		panic("face surface: expected a plane")
	}
	fmt.Printf("%s origin=%v normal=%v u=%v v=%v\n", hex.EncodeToString(face.ReferenceKey()), plane.Origin, plane.Normal(), plane.UAxis, plane.VAxis)
}
func main() {
	if len(os.Args) != 2 {
		panic("arguments: expected a cube fixture path")
	}
	workspace := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	document, err := workspace.Open(os.Args[1], true)
	if err != nil {
		panic(err)
	}
	part := document.Content().(*compdef.PartComponentDefinition)
	bodies := part.SurfaceBodies().All()
	if len(bodies) != 1 {
		panic(fmt.Sprintf("bodies %d: expected one", len(bodies)))
	}
	for _, face := range bodies[0].Faces() {
		printFace(face)
	}
}
