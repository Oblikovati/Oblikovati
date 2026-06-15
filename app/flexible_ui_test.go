// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
)

// TestOccurrenceFlexibleMenu checks the M12-F06 Flexible toggle: only a sub-assembly
// occurrence's menu offers it, invoking it sets the flag, and the label then reads Rigid (#822).
func TestOccurrenceFlexibleMenu(t *testing.T) {
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	leaf := asm.Occurrences().Item(0)

	if _, ok := findMenuItem(occurrenceMenu(OccurrenceHandle{Occurrence: leaf}), "Flexible"); ok {
		t.Error("a leaf part occurrence's menu should not offer Flexible")
	}

	sub := asm.Place("sub:1", compdef.NewAssemblyComponentDefinition(), math.Identity4())
	item, ok := findMenuItem(occurrenceMenu(OccurrenceHandle{Occurrence: sub}), "Flexible")
	if !ok {
		t.Fatal("a sub-assembly occurrence's menu should offer Flexible")
	}
	if err := item.Invoke(s); err != nil {
		t.Fatalf("Flexible: %v", err)
	}
	if !sub.Flexible() {
		t.Error("invoking Flexible did not set the flag")
	}
	if _, ok := findMenuItem(occurrenceMenu(OccurrenceHandle{Occurrence: sub}), "Rigid"); !ok {
		t.Error("a flexible occurrence's menu should offer Rigid to turn it off")
	}
}
