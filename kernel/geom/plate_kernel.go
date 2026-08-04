// SPDX-License-Identifier: GPL-2.0-only

package geom

import stdmath "math"

// The Duchon order-m=3 polyharmonic fundamental solution in d=2 and its partials up to order
// 2, transcribed verbatim from .superpowers/sdd/plate-math-kit.md §1 (numerically confirmed
// there against central finite differences). We commit to OCCT's SolEm convention — NOT the
// textbook φ=r⁴log r, which is E/2 — and use it for E and EVERY partial, so the RBF part is
// never silently doubled or halved. With R = r² = Δu²+Δv² (the kit's "R", the squared radius):
//
//	E     = R²·log R                       (= r⁴·log(r²) = 2·r⁴·log r)
//	E_u   = 2·Δu·R·(2·log R + 1)
//	E_uu  = 2·R·(2 log R + 1) + 4·Δu²·(2 log R + 3)
//	E_uv  = 4·Δu·Δv·(2 log R + 3)
//
// (E_v, E_vv symmetric in Δv.) Every partial → 0 as r→0 (removable singularity); the rFloor
// guard returns that limit outright rather than log-ing a clamped R (which would leak a
// spurious nonzero value at a self-term P_i−P_i=0). rFloor = (res.Weld·L)² per the kit.

// plateRadial returns R = Δu²+Δv² and log R, flagging the removable-singularity region
// R < rFloor where E and all its partials take their r→0 limit of 0. Pulled out so the six
// kernel functions share one floor guard and one log (no duplication, kit §1's "guard on
// R_floor, else lr = log R and combine").
func plateRadial(du, dv, rFloor float64) (bigR, logR float64, singular bool) {
	bigR = du*du + dv*dv
	if bigR < rFloor {
		return bigR, 0, true
	}
	return bigR, stdmath.Log(bigR), false
}

// plateE is the fundamental solution E = R²·log R (OCCT SolEm convention; kit §1).
func plateE(du, dv, rFloor float64) float64 {
	bigR, logR, singular := plateRadial(du, dv, rFloor)
	if singular {
		return 0
	}
	return bigR * bigR * logR
}

// plateEu is ∂E/∂Δu = 2·Δu·R·(2·log R + 1) — "the single most load-bearing derivative"
// (kit §1): the RBF entry wherever a G0 row meets a G1 column. Odd in Δu.
func plateEu(du, dv, rFloor float64) float64 {
	bigR, logR, singular := plateRadial(du, dv, rFloor)
	if singular {
		return 0
	}
	return 2 * du * bigR * (2*logR + 1)
}

// plateEv is ∂E/∂Δv = 2·Δv·R·(2·log R + 1) — the Δv companion of plateEu. Odd in Δv.
func plateEv(du, dv, rFloor float64) float64 {
	bigR, logR, singular := plateRadial(du, dv, rFloor)
	if singular {
		return 0
	}
	return 2 * dv * bigR * (2*logR + 1)
}

// plateEuu is ∂²E/∂Δu² = 2·R·(2 log R + 1) + 4·Δu²·(2 log R + 3) (kit §1); appears wherever a
// ∂u row meets a ∂u column (combined order 2). Even.
func plateEuu(du, dv, rFloor float64) float64 {
	bigR, logR, singular := plateRadial(du, dv, rFloor)
	if singular {
		return 0
	}
	return 2*bigR*(2*logR+1) + 4*du*du*(2*logR+3)
}

// plateEvv is ∂²E/∂Δv² = 2·R·(2 log R + 1) + 4·Δv²·(2 log R + 3) — the Δv companion of
// plateEuu. Even.
func plateEvv(du, dv, rFloor float64) float64 {
	bigR, logR, singular := plateRadial(du, dv, rFloor)
	if singular {
		return 0
	}
	return 2*bigR*(2*logR+1) + 4*dv*dv*(2*logR+3)
}

// plateEuv is ∂²E/∂Δu∂Δv = 4·Δu·Δv·(2 log R + 3) (kit §1); the ∂u-row/∂v-column entry. Even.
func plateEuv(du, dv, rFloor float64) float64 {
	_, logR, singular := plateRadial(du, dv, rFloor)
	if singular {
		return 0
	}
	return 4 * du * dv * (2*logR + 3)
}

// plateDeriv evaluates D^{(a,b)}E at (Δu,Δv) for combined order a+b ≤ 2 — the only orders the
// bordered assembly (K, combined row+column order ≤ 2) and EvalGrad (constraint order ≤ 1,
// plus one differentiation) ever ask for. Dispatches to the six closed forms above.
func plateDeriv(a, b int, du, dv, rFloor float64) float64 {
	switch {
	case a == 0 && b == 0:
		return plateE(du, dv, rFloor)
	case a == 1 && b == 0:
		return plateEu(du, dv, rFloor)
	case a == 0 && b == 1:
		return plateEv(du, dv, rFloor)
	case a == 2 && b == 0:
		return plateEuu(du, dv, rFloor)
	case a == 0 && b == 2:
		return plateEvv(du, dv, rFloor)
	default: // a == 1 && b == 1
		return plateEuv(du, dv, rFloor)
	}
}
