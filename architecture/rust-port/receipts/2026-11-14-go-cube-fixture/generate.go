package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	stdmath "math"
	"os"
	"sort"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	m "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
	"oblikovati.org/persistence"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func main() {
	if len(os.Args) != 2 {
		panic("expected output file")
	}
	path := os.Args[1]
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	d, err := compdef.AddPart(ws, path, true)
	must(err)
	part := d.Content().(*compdef.PartComponentDefinition)
	size, err := part.Parameters().AddUserParameter("size", "10 mm")
	must(err)
	s := part.Sketches().Add(sketch.XYPlane())
	p0 := s.Points().Add(m.P2(0, 0))
	p1 := s.Points().Add(m.P2(1, 0))
	p2 := s.Points().Add(m.P2(1, 1))
	p3 := s.Points().Add(m.P2(0, 1))
	s.Lines().Add(p0, p1)
	s.Lines().Add(p1, p2)
	s.Lines().Add(p2, p3)
	s.Lines().Add(p3, p0)
	g := s.GeometricConstraints()
	g.AddFix(p0)
	g.AddHorizontal(p0, p1)
	g.AddVertical(p1, p2)
	g.AddHorizontal(p2, p3)
	g.AddVertical(p3, p0)
	_, err = s.DimensionConstraints().AddDistance(p0, p1, "size")
	must(err)
	_, err = s.DimensionConstraints().AddDistance(p1, p2, "size")
	must(err)
	solved := s.Solve()
	if !solved.Converged || solved.DOF != 0 {
		panic(fmt.Sprintf("sketch solve: %+v", solved))
	}
	feature.NewExtrudeFeatures(part.Features()).AddByDistanceExtent(s, 0, ops.NewBody, func() float64 { return size.ModelValue() })
	part.Recompute()
	bodies := part.SurfaceBodies().All()
	if len(bodies) != 1 {
		panic(fmt.Sprintf("bodies %d", len(bodies)))
	}
	keys := make([]string, 0)
	for _, f := range bodies[0].Faces() {
		area, exact := topo.NewFaceEvaluator(f).Area()
		if !exact || stdmath.Abs(area-1) > 1e-9 {
			panic(fmt.Sprintf("face area %g, exact=%v", area, exact))
		}
		keys = append(keys, hex.EncodeToString(f.ReferenceKey()))
	}
	sort.Strings(keys)
	if len(keys) != 6 {
		panic(fmt.Sprintf("faces %d", len(keys)))
	}
	for i, k := range keys {
		if k == "" || (i > 0 && k == keys[i-1]) {
			panic("empty or duplicate face reference")
		}
	}
	must(ws.Save(d))
	raw, err := os.ReadFile(path)
	must(err)
	reopened, err := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default()).Open(path, true)
	must(err)
	replay := reopened.Content().(*compdef.PartComponentDefinition)
	replayBodies := replay.SurfaceBodies().All()
	if len(replayBodies) != 1 {
		panic(fmt.Sprintf("replay bodies %d", len(replayBodies)))
	}
	got := make([]string, 0)
	for _, f := range replayBodies[0].Faces() {
		got = append(got, hex.EncodeToString(f.ReferenceKey()))
	}
	sort.Strings(got)
	if !bytes.Equal([]byte(fmt.Sprint(keys)), []byte(fmt.Sprint(got))) {
		panic(fmt.Sprintf("face keys differ %v %v", keys, got))
	}
	changed, ok := replay.Parameters().ByName("size")
	if !ok {
		panic("size parameter missing after reopen")
	}
	must(replay.Parameters().SetExpression(changed.ID(), "20 mm"))
	again := replay.Sketches().Item(0).Solve()
	if !again.Converged || again.DOF != 0 {
		panic(fmt.Sprintf("replay solve: %+v", again))
	}
	replay.Features().MarkDirty(replay.Features().Item(0))
	replay.Recompute()
	box := replay.RangeBox().Diagonal()
	if stdmath.Abs(float64(box.X)-2) > 1e-9 || stdmath.Abs(float64(box.Y)-2) > 1e-9 || stdmath.Abs(float64(box.Z)-1) > 1e-9 {
		panic(fmt.Sprintf("unexpected edited range %v", box))
	}
	fmt.Fprintf(os.Stderr, "after size edit: %.9g %.9g %.9g cm\n", box.X, box.Y, box.Z)
	data := map[string]any{"dof": solved.DOF, "face_keys_hex": keys, "reopened_size_20mm_range_cm": []float64{float64(box.X), float64(box.Y), float64(box.Z)}, "face_area_cm2": 1, "document_bytes": len(raw)}
	b, err := json.MarshalIndent(data, "", "  ")
	must(err)
	fmt.Println(string(b))
}
