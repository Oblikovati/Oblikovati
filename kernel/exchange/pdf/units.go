// SPDX-License-Identifier: GPL-2.0-only

package pdf

// mmPerPoint converts a PDF user-space unit (1/72 inch) to millimetres. The decoder
// bakes this factor into every emitted coordinate and tags the drawing as millimetres
// (drawing.INSMillimetres) so the shared importer scales it into the document's
// database unit like any other drawing.
const mmPerPoint = 25.4 / 72.0

// toMM converts a coordinate in PDF points to millimetres.
func toMM(pt float64) float64 { return pt * mmPerPoint }
