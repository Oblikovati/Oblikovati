// SPDX-License-Identifier: GPL-2.0-only

package param

// ISO 286 limits-and-fits reference data (#1848). The standard tolerance grade
// widths (IT) are the authoritative published µm values, indexed by nominal-size
// step; the fundamental-deviation formulas below reproduce the tabulated shaft
// deviations. Together they resolve the same bands the ISO 286-2 fit tables list
// (validated against 50H7/g6, 50f7, 50d9, 25f7, 10g6, … in iso286_test.go).
//
// Scope: nominal sizes over 3 mm up to and including 500 mm (where the size
// step's geometric mean D = √(lo·hi) is unambiguous), grades IT5–IT11, and the
// clearance/location letters d,f,g,h (with the hole letters D,F,G,H by the
// EI = −es rule) plus symmetric js/JS. Anything outside this range is rejected
// with a clear error rather than returning a dubious band.

// iso286FirstGrade / iso286LastGrade bound the supported IT grades.
const (
	iso286FirstGrade = 5
	iso286LastGrade  = 11
)

// iso286StepUpper[i] is the upper bound (mm, inclusive) of nominal-size step i;
// iso286StepLower[i] its exclusive lower bound. Step 0 (0–3 mm) is unsupported
// for fits (its geometric mean is ill-defined) but kept so indices line up with
// the published table rows.
var (
	iso286StepUpper = []float64{3, 6, 10, 18, 30, 50, 80, 120, 180, 250, 315, 400, 500}
	iso286StepLower = []float64{0, 3, 6, 10, 18, 30, 50, 80, 120, 180, 250, 315, 400}
)

// iso286IT holds the standard tolerance grade widths in µm, rows indexed by
// nominal-size step (matching iso286StepUpper), columns IT5..IT11. These are the
// published ISO 286-1 values.
var iso286IT = [][]int{
	//  IT5  IT6  IT7  IT8  IT9  IT10 IT11
	{4, 6, 10, 14, 25, 40, 60},      // 0–3
	{5, 8, 12, 18, 30, 48, 75},      // 3–6
	{6, 9, 15, 22, 36, 58, 90},      // 6–10
	{8, 11, 18, 27, 43, 70, 110},    // 10–18
	{9, 13, 21, 33, 52, 84, 130},    // 18–30
	{11, 16, 25, 39, 62, 100, 160},  // 30–50
	{13, 19, 30, 46, 74, 120, 190},  // 50–80
	{15, 22, 35, 54, 87, 140, 220},  // 80–120
	{18, 25, 40, 63, 100, 160, 250}, // 120–180
	{20, 29, 46, 72, 115, 185, 290}, // 180–250
	{23, 32, 52, 81, 130, 210, 320}, // 250–315
	{25, 36, 57, 89, 140, 230, 360}, // 315–400
	{27, 40, 63, 97, 155, 250, 400}, // 400–500
}
