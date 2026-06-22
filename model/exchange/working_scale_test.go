// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
)

// TestWorkingUnitMMTracksDocumentScale is the ADR-0042 Phase 2 exchange boundary (#1248): the
// translators' database-unit size in millimetres follows the document's working scale, so a
// document centred on a non-cm working unit exports/imports at the correct scale. A cm document
// (working scale 1) yields the historical 10 mm — unchanged.
func TestWorkingUnitMMTracksDocumentScale(t *testing.T) {
	part := compdef.NewPartComponentDefinition()

	if got := workingUnitMM(part); got != exchange.DBUnitMM {
		t.Errorf("cm document workingUnitMM = %v, want %v (10 mm)", got, exchange.DBUnitMM)
	}
	if opts := exportUnits(part); opts.TargetUnitMM != exchange.DBUnitMM {
		t.Errorf("cm document export TargetUnitMM = %v, want %v", opts.TargetUnitMM, exchange.DBUnitMM)
	}

	// Centre the document on micrometres: 1 working unit = 1 µm = 1e-4 cm ⇒ 1e-3 mm.
	um, err := part.Units().CenteredOnLength("µm")
	if err != nil {
		t.Fatal(err)
	}
	part.SetUnits(um)
	if got, want := workingUnitMM(part), 1e-4*exchange.DBUnitMM; got != want {
		t.Errorf("µm document workingUnitMM = %v, want %v (1e-3 mm)", got, want)
	}
	if opts := exportUnits(part); opts.TargetUnitMM != 1e-4*exchange.DBUnitMM {
		t.Errorf("µm document export TargetUnitMM = %v, want %v", opts.TargetUnitMM, 1e-4*exchange.DBUnitMM)
	}
	// The export still declares the document's preferred display unit as the file unit.
	if u := exportUnits(part).FileUnit; u != part.Units().PreferredName(param.Length) {
		t.Errorf("export FileUnit = %q, want the document preferred unit %q", u, part.Units().PreferredName(param.Length))
	}
}
