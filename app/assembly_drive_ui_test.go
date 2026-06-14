// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/assembly"
)

// TestJointsPanelExposesDrive: the Joints panel carries the Drive command as a compact icon
// button (M12-F03).
func TestJointsPanelExposesDrive(t *testing.T) {
	tab, ok := BuildRibbon(assemblySession(t)).Tab("Assemble")
	if !ok {
		t.Fatal("an active assembly should show the Assemble tab")
	}
	panel, ok := tab.Panel("Joints")
	if !ok {
		t.Fatal("Assemble tab has no Joints panel")
	}
	if !hasButton(panel, "Drive") {
		t.Fatal("Joints panel is missing the Drive command")
	}
	if got, ok := styleOf(panel, "Drive"); !ok || got != CompactIconButton {
		t.Errorf("Drive button style = %v, want CompactIconButton", got)
	}
}

// TestDefaultDriveSettingsByKind: a rotational joint sweeps its angular range (a full turn
// when unbounded), a slider its linear range.
func TestDefaultDriveSettingsByKind(t *testing.T) {
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	placedWidget(t, s, asm, "widget:2")
	a, b := asm.Occurrences().Item(0), asm.Occurrences().Item(1)
	z, _ := math.NewUnitVector3(0, 0, 1)
	rot := asm.Joints().AddRotational(
		assembly.Ref{Occurrence: a, Primitive: assembly.LinePrimitive(math.P3(0, 0, 0), z)},
		assembly.Ref{Occurrence: b, Primitive: assembly.LinePrimitive(math.P3(0, 0, 0), z)})
	slide := asm.Joints().AddSlider(
		assembly.Ref{Occurrence: a, Primitive: assembly.LinePrimitive(math.P3(0, 0, 0), z)},
		assembly.Ref{Occurrence: b, Primitive: assembly.LinePrimitive(math.P3(0, 0, 0), z)})

	rs := defaultDriveSettings(rot)
	if rs.Variable() != types.DriveAngular {
		t.Errorf("rotational drive variable = %s, want angular", rs.Variable())
	}
	if _, end := rs.Range(); stdmath.Abs(end-2*stdmath.Pi) > 1e-9 {
		t.Errorf("rotational drive end = %.4f, want 2π", end)
	}
	if ss := defaultDriveSettings(slide); ss.Variable() != types.DriveLinear {
		t.Errorf("slider drive variable = %s, want linear", ss.Variable())
	}
}

// TestSelectedDrivableJointGate: Drive enables only when a drivable joint is selected.
func TestSelectedDrivableJointGate(t *testing.T) {
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	placedWidget(t, s, asm, "widget:2")
	a, b := asm.Occurrences().Item(0), asm.Occurrences().Item(1)
	z, _ := math.NewUnitVector3(0, 0, 1)
	rot := asm.Joints().AddRotational(
		assembly.Ref{Occurrence: a, Primitive: assembly.LinePrimitive(math.P3(0, 0, 0), z)},
		assembly.Ref{Occurrence: b, Primitive: assembly.LinePrimitive(math.P3(0, 0, 0), z)})

	if hasSelectedDrivableJoint(s) {
		t.Error("no selection: Drive should be disabled")
	}
	s.Selection().Add(AssemblyJointHandle{Joint: rot})
	if !hasSelectedDrivableJoint(s) {
		t.Error("rotational joint selected: Drive should be enabled")
	}
}

// TestDrivePlaybackMovesThenRestores: starting playback and ticking moves the driven
// component; stopping restores its pre-drive pose.
func TestDrivePlaybackMovesThenRestores(t *testing.T) {
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	placedWidget(t, s, asm, "widget:2")
	a, b := asm.Occurrences().Item(0), asm.Occurrences().Item(1)
	a.SetGrounded(true)
	z, _ := math.NewUnitVector3(0, 0, 1)
	rot := asm.Joints().AddRotational(
		assembly.Ref{Occurrence: a, Primitive: assembly.LinePrimitive(math.P3(0, 0, 0), z)},
		assembly.Ref{Occurrence: b, Primitive: assembly.LinePrimitive(math.P3(0, 0, 0), z)})

	rest := b.Transform()
	if err := s.StartDriveAnimation(rot.ID(), defaultDriveSettings(rot)); err != nil {
		t.Fatalf("StartDriveAnimation: %v", err)
	}
	if !s.DriveAnimating() {
		t.Fatal("DriveAnimating should be true after start")
	}
	s.TickDriveAnimation(driveFrameSeconds * 6) // advance to a rotated frame
	if b.Transform().IsEqualTo(rest, 1e-9) {
		t.Error("driven component did not move during playback")
	}
	s.StopDriveAnimation()
	if s.DriveAnimating() {
		t.Error("DriveAnimating should be false after stop")
	}
	if !b.Transform().IsEqualTo(rest, 1e-9) {
		t.Error("stopping playback did not restore the rest pose")
	}
}
