// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import "oblikovati.org/kernel/geom"

// WorstEdgeChord exposes the achieved chord measurement to the corpus rows that pin it.
func WorstEdgeChord(m *Mesh, s geom.Surface) float64 { return worstEdgeChord(m, s) }
