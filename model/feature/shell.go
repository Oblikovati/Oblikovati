// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
)

// shellDirectionNames is the stable wire/serialization vocabulary for ops.ShellDirection —
// the single source shared by the op registry (wire) and the .obk codec so they cannot drift.
var shellDirectionNames = map[ops.ShellDirection]string{
	ops.ShellInside:  "inside",
	ops.ShellOutside: "outside",
	ops.ShellBoth:    "both",
}

// ShellDirectionName renders a shell direction as its stable name ("" for the inside default, so
// the common case serializes nothing).
func ShellDirectionName(d ops.ShellDirection) string {
	if d == ops.ShellInside {
		return ""
	}
	return shellDirectionNames[d]
}

// ParseShellDirection maps a name ("inside"/"outside"/"both", empty ⇒ inside) to its direction;
// ok is false for an unknown name.
func ParseShellDirection(name string) (ops.ShellDirection, bool) {
	switch name {
	case "", "inside":
		return ops.ShellInside, true
	case "outside":
		return ops.ShellOutside, true
	case "both":
		return ops.ShellBoth, true
	default:
		return ops.ShellInside, false
	}
}

// evalFaceThicknesses resolves the per-face wall overrides (#1864) to the plain values the kernel
// takes, so a parameter change re-evaluates them like every other driven dimension.
func evalFaceThicknesses(fts []ShellFaceThickness) []ops.ShellFaceThickness {
	out := make([]ops.ShellFaceThickness, 0, len(fts))
	for _, ft := range fts {
		out = append(out, ops.ShellFaceThickness{FaceKey: ft.FaceKey, Thickness: callOrZero(ft.Thickness)})
	}
	return out
}

// shellBody hollows the running body to a wall thickness on the chosen side (inside/outside/both),
// opening the removed faces, via ops.ShellVaried, and replaces it in the body list. A lost face
// key or non-positive thickness is an error so the feature goes Sick. See kernel/ops/shell.go.
func shellBody(in Input, removedFaceKeys [][]byte, thickness float64, dir ops.ShellDirection,
	faceThicknesses []ops.ShellFaceThickness, feat string) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	if thickness <= 0 {
		return Output{}, fmt.Errorf("%s: thickness %g must be > 0", feat, thickness)
	}
	result, err := ops.ShellVaried(body, removedFaceKeys, thickness, dir, faceThicknesses)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceBody(in.Bodies, body, result)}, nil
}
