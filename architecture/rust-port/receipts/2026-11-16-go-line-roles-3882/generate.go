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

func must(err error) {
	if err != nil {
		panic(err)
	}
}

type observation struct {
	ID                    int  `json:"id"`
	Centerline            bool `json:"centerline"`
	EffectiveConstruction bool `json:"effectiveConstruction"`
	ExplicitConstruction  bool `json:"explicitConstruction"`
}

func main() {
	if len(os.Args) != 2 {
		panic("expected output path")
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	document, err := compdef.AddPart(ws, os.Args[1], true)
	must(err)
	part := document.Content().(*compdef.PartComponentDefinition)
	s := part.Sketches().Add(sketch.XYPlane())
	for i := 0; i < 4; i++ {
		a := s.Points().Add(m.P2(float64(i), 0))
		b := s.Points().Add(m.P2(float64(i), 1))
		line := s.Lines().Add(a, b)
		line.SetConstruction(i&1 != 0)
		line.SetCenterline(i&2 != 0)
	}
	must(ws.Save(document))
	reopened, err := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default()).Open(os.Args[1], true)
	must(err)
	restored := reopened.Content().(*compdef.PartComponentDefinition).Sketches().Item(0)
	var rows []observation
	for i := 0; i < restored.Lines().Count(); i++ {
		line := restored.Lines().Item(i)
		centerline := line.IsCenterline()
		effective := line.IsConstruction()
		line.SetCenterline(false)
		rows = append(rows, observation{int(line.EntityID()), centerline, effective, line.IsConstruction()})
	}
	must(json.NewEncoder(os.Stdout).Encode(rows))
}
