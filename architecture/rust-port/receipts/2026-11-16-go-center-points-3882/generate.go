// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"encoding/json"
	m "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/sketch"
	"oblikovati.org/persistence"
	"os"
)

type observation struct {
	ID         int  `json:"id"`
	Center     bool `json:"center"`
	Standalone bool `json:"standalone"`
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func main() {
	if len(os.Args) != 2 {
		panic("expected output path")
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	document, err := compdef.AddPart(ws, os.Args[1], true)
	must(err)
	s := document.Content().(*compdef.PartComponentDefinition).Sketches().Add(sketch.XYPlane())
	s.Points().Add(m.P2(0, 0)).SetCenterPoint(true)
	s.Points().Add(m.P2(2, 0))
	line := s.Lines().AddByTwoPoints(m.P2(0, 1), m.P2(2, 1))
	line.A.SetCenterPoint(true)
	must(ws.Save(document))
	reopened, err := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default()).Open(os.Args[1], true)
	must(err)
	restored := reopened.Content().(*compdef.PartComponentDefinition).Sketches().Item(0)
	points := []*sketch.Point{restored.Points().Item(0), restored.Points().Item(1), restored.Lines().Item(0).A, restored.Lines().Item(0).B}
	var rows []observation
	for i, point := range points {
		rows = append(rows, observation{int(point.EntityID()), point.IsCenterPoint(), i < 2})
	}
	must(json.NewEncoder(os.Stdout).Encode(rows))
}
