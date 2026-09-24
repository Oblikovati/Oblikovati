// SPDX-License-Identifier: GPL-2.0-only

package geom

// The torus section's TOPOLOGY oracle (ADR-0066, Oblikovati#3514).
//
// "A fast path must return the same topology the general path would return" is only a measurement if
// something counts the topology by a route the fast path does not share. This is that route: the number
// of connected components of the section on the chart, read from the SIGN of the station polynomial on
// a grid and nothing else — no root solving, no lane pairing, no window finding, no fold bisection.
// torus_torus_test.go's corpus rows assert the kernel's curve count against it.

// torusSectionComponents counts the CONNECTED COMPONENTS of the section on the chart, from the sign of
// the station polynomial on a (u, v) grid alone — no root solving, no lane pairing, no window finding.
// It is the topology oracle: the number of closed curves a correct section has, arrived at by a route
// that shares nothing with the reduction beyond the five coefficients themselves.
//
// A grid CELL is on the section when the polynomial does not keep one sign across its four corners.
// Such cells are joined to their neighbours on the torus's doubly periodic grid, and the number of
// classes is the number of section curves. A pair whose contact is thinner than one cell is invisible
// to it, which is why the corpus rows are ordinary contacts rather than grazes.
func torusSectionComponents(chart Torus, co TorusCoForm) int {
	sign := torusStationSignGrid(chart, co)
	parent := make([]int, torusComponentGrid*torusComponentGrid)
	for i := range parent {
		parent[i] = i
	}
	cut := make([]bool, len(parent))
	for i := range torusComponentGrid {
		for j := range torusComponentGrid {
			cut[i*torusComponentGrid+j] = torusCellIsCut(sign, i, j)
		}
	}
	joinCutNeighbours(parent, cut)
	return countCutClasses(parent, cut)
}

// torusStationSignGrid samples the sign of f over the chart's (u, v) grid: true where the co-form's
// implicit value is positive.
func torusStationSignGrid(chart Torus, co TorusCoForm) [][]bool {
	out := make([][]bool, torusComponentGrid)
	for i := range torusComponentGrid {
		v := float64(twoPi * float64(i) / torusComponentGrid)
		poly := co.stationOn(chart, v).secondHarmonic()
		out[i] = make([]bool, torusComponentGrid)
		for j := range torusComponentGrid {
			out[i][j] = poly.valueAt(float64(twoPi*float64(j)/torusComponentGrid)) > 0
		}
	}
	return out
}

// torusCellIsCut reports the cell with corner (i, j) carrying both signs, so the section crosses it.
func torusCellIsCut(sign [][]bool, i, j int) bool {
	n := torusComponentGrid
	a := sign[i][j]
	return a != sign[(i+1)%n][j] || a != sign[i][(j+1)%n] || a != sign[(i+1)%n][(j+1)%n]
}

// joinCutNeighbours unions every cut cell with its cut neighbours, wrapping both ways.
func joinCutNeighbours(parent []int, cut []bool) {
	n := torusComponentGrid
	for i := range n {
		for j := range n {
			if !cut[i*n+j] {
				continue
			}
			for _, nb := range [][2]int{{(i + 1) % n, j}, {i, (j + 1) % n}} {
				if cut[nb[0]*n+nb[1]] {
					unionCells(parent, i*n+j, nb[0]*n+nb[1])
				}
			}
		}
	}
}

// countCutClasses is how many distinct roots the cut cells resolve to.
func countCutClasses(parent []int, cut []bool) int {
	seen := map[int]bool{}
	for i, isCut := range cut {
		if isCut {
			seen[findCell(parent, i)] = true
		}
	}
	return len(seen)
}

// findCell and unionCells are the union-find the component count runs on.
func findCell(parent []int, i int) int {
	for parent[i] != i {
		parent[i] = parent[parent[i]]
		i = parent[i]
	}
	return i
}

func unionCells(parent []int, a, b int) {
	ra, rb := findCell(parent, a), findCell(parent, b)
	if ra != rb {
		parent[ra] = rb
	}
}

// torusComponentGrid is the oracle's grid side. At 512 a cell is a hundredth of a radian, which
// resolves every contact in the corpus and costs a few million polynomial evaluations.
const torusComponentGrid = 512
