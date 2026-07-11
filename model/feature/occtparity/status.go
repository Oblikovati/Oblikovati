// SPDX-License-Identifier: GPL-2.0-only

package occtparity

// Outcome is one case's verdict, in OCCT's own vocabulary (parse.rules), so our scoreboard
// reads the same way OCCT's own test-summary does.
type Outcome int

const (
	Pass                 Outcome = iota
	FailFaulty                   // result not a valid solid (OCCT \bFaulty\b)
	FailArea                     // valid but area outside tolerance (asserted separately)
	Incomplete                   // OCCT tolerance-ang IGNORE analogue
	SkipTODO                     // OCCT TODO/INCOMPLETE marker present
	SkipImportDivergence         // STEP input did not import faithfully — not a fillet defect
)

// String names an outcome for the scoreboard table.
func (o Outcome) String() string {
	switch o {
	case Pass:
		return "PASS"
	case FailFaulty:
		return "FAIL(faulty)"
	case FailArea:
		return "FAIL(area)"
	case Incomplete:
		return "SKIP(varradius)"
	case SkipTODO:
		return "SKIP(todo)"
	case SkipImportDivergence:
		return "SKIP(import)"
	default:
		return "UNKNOWN"
	}
}

// classify maps one case's run facts to OCCT's verdict semantics. TODO wins (we never claim
// to be stricter than OCCT); import divergence is separated from fillet defects so a STEP
// round-trip gap never gets blamed on the fillet engine; an invalid result is Faulty.
//
// Example:
//
//	classify(r, true, true, true) // == Pass
func classify(r Record, importOK, filletOK, valid bool) Outcome {
	switch {
	case r.TODO != "":
		return SkipTODO
	case !importOK:
		return SkipImportDivergence
	case !filletOK || !valid:
		return FailFaulty
	default:
		return Pass
	}
}
