// SPDX-License-Identifier: GPL-2.0-only

package modelaccess

import (
	"errors"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
)

func TestActivePart(t *testing.T) {
	// No active document.
	if _, err := ActivePart(app.NewSession()); !errors.Is(err, ErrNoActiveDocument) {
		t.Fatalf("ActivePart(empty) = %v, want ErrNoActiveDocument", err)
	}

	// A realized part document.
	s := app.NewSession()
	d, err := s.Workspace().Add(doc.Part, "p.obk", true)
	if err != nil {
		t.Fatal(err)
	}
	def := compdef.NewPartComponentDefinition()
	d.SetContent(def)
	if got, err := ActivePart(s); err != nil || got != def {
		t.Fatalf("ActivePart(part) = %v, %v; want the part definition", got, err)
	}

	// The active document is not a part.
	as := app.NewSession()
	ad, _ := as.Workspace().Add(doc.Assembly, "a.obk", true)
	ad.SetContent(compdef.NewAssemblyComponentDefinition())
	if _, err := ActivePart(as); err == nil {
		t.Fatal("ActivePart(assembly) = nil error, want a not-a-part error")
	}
}

func TestActiveAssembly(t *testing.T) {
	if _, err := ActiveAssembly(app.NewSession()); !errors.Is(err, ErrNoActiveDocument) {
		t.Fatalf("ActiveAssembly(empty) = %v, want ErrNoActiveDocument", err)
	}

	s := app.NewSession()
	d, err := s.Workspace().Add(doc.Assembly, "a.obk", true)
	if err != nil {
		t.Fatal(err)
	}
	def := compdef.NewAssemblyComponentDefinition()
	d.SetContent(def)
	if got, err := ActiveAssembly(s); err != nil || got != def {
		t.Fatalf("ActiveAssembly(assembly) = %v, %v; want the assembly definition", got, err)
	}

	ps := app.NewSession()
	pd, _ := ps.Workspace().Add(doc.Part, "p.obk", true)
	pd.SetContent(compdef.NewPartComponentDefinition())
	if _, err := ActiveAssembly(ps); err == nil {
		t.Fatal("ActiveAssembly(part) = nil error, want a not-an-assembly error")
	}
}
