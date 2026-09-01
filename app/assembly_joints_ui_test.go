// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/assembly"
)

// TestJointsPanelExposesEveryJoint: the Assemble tab's Joints panel exposes one command per
// M12-F02 joint kind, each a compact icon button.
func TestJointsPanelExposesEveryJoint(t *testing.T) {
	t.Parallel()
	tab, ok := BuildRibbon(assemblySession(t)).Tab("Assemble")
	if !ok {
		t.Fatal("an active assembly should show the Assemble tab")
	}
	panel, ok := tab.Panel("Joints")
	if !ok {
		t.Fatal("Assemble tab has no Joints panel")
	}
	for _, name := range []string{"Rigid", "Rotational", "Slider", "Cylindrical", "Planar", "Ball"} {
		if !hasButton(panel, name) {
			t.Errorf("Joints panel is missing the %q command", name)
		}
		if got, ok := styleOf(panel, name); !ok || got != CompactIconButton {
			t.Errorf("%q button style = %v, want CompactIconButton", name, got)
		}
	}
}

// TestAssemblyBrowserListsJointsInOrder: an assembly's joints appear under a Joints folder in
// creation order, each a selectable row carrying an AssemblyJointHandle (M12-F02).
func TestAssemblyBrowserListsJointsInOrder(t *testing.T) {
	t.Parallel()
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	placedWidget(t, s, asm, "widget:2")
	a := asm.Occurrences().Item(0)
	b := asm.Occurrences().Item(1)
	z, _ := math.NewUnitVector3(0, 0, 1)
	rot := asm.Joints().AddRotational(
		assembly.Ref{Occurrence: a, Primitive: assembly.LinePrimitive(math.P3(0, 0, 0), z)},
		assembly.Ref{Occurrence: b, Primitive: assembly.LinePrimitive(math.P3(0, 0, 0), z)})
	slide := asm.Joints().AddSlider(
		assembly.Ref{Occurrence: a, Primitive: assembly.LinePrimitive(math.P3(0, 0, 0), z)},
		assembly.Ref{Occurrence: b, Primitive: assembly.LinePrimitive(math.P3(0, 0, 0), z)})

	folder := findBrowserNode(BuildBrowser(s), "joints", "Joints")
	if folder == nil {
		t.Fatal("assembly browser has no Joints folder")
	}
	if len(folder.Children) != 2 || folder.Children[0].Label != rot.Name() || folder.Children[1].Label != slide.Name() {
		t.Fatalf("joint rows = %d, want [%q,%q] in creation order", len(folder.Children), rot.Name(), slide.Name())
	}
	h, ok := folder.Children[0].Select.(AssemblyJointHandle)
	if !ok || h.Joint.ID() != rot.ID() {
		t.Errorf("first row selects %T, want the rotational joint's AssemblyJointHandle", folder.Children[0].Select)
	}
}

// TestAssemblyJointToolShell checks the joint tool gathers two face picks and rejects an
// unresolved commit cleanly.
func TestAssemblyJointToolShell(t *testing.T) {
	t.Parallel()
	tool := NewAssemblyJointTool("Rotational", func(js *assembly.JointSet, r []assembly.Ref) assembly.Joint {
		return js.AddRotational(r[0], r[1])
	})
	if tool.CanCommit() {
		t.Error("no picks: CanCommit should be false")
	}
	tool.Pick(nil, FaceHandle{})
	tool.Pick(nil, FaceHandle{})
	if !tool.CanCommit() {
		t.Error("two picks: CanCommit should be true")
	}
	if err := tool.Commit(assemblySession(t)); err == nil {
		t.Error("commit with unresolved faces should error")
	}
}
