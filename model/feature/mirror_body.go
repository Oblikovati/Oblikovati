// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Mirroring a whole solid rather than a set of features (Oblikovati#1890) — Inventor's
// MirrorFeature.MirrorOfBody.
//
// Feature mode re-applies the source features' own tools on the far side of the plane, which needs
// those features to exist and to be replicable. Body mode reflects the RUNNING SOLID instead, and
// that is how most symmetric parts are actually built: model one half, mirror it, join. It also
// works when the half was imported or built by features a pattern cannot replicate, since it never
// looks at the recipe.
//
// The two options Inventor allows only in body mode are honoured only in body mode here too:
//
//   - RemoveOriginal keeps just the reflection ("this property only applies if MirrorOfBody is
//     True"), which is how a left-handed part is made from a right-handed one.
//   - Operation is kNewBodyOperation or kJoinOperation ("only applied when MirrorOfBody is True"):
//     leave the reflection a separate solid, or union it with the original into one. It is carried
//     as JoinToOriginal — a bool — and not as an ops.PartFeatureOperation, because ops.Join is that
//     enum's ZERO value: a definition built without naming an operation would silently join.
//
// A reflection has a negative determinant, so it turns a solid inside out; transform.TransformBody
// already detects that and flips face senses, which is what keeps the mirrored half's volume
// positive and its tessellation wound outward.

// mirrorBodies reflects every running body across the plane and combines the halves as the
// definition asks.
func (m *MirrorFeature) mirrorBodies(in Input, xf math.Matrix4) (Output, error) {
	if len(in.Bodies) == 0 {
		return Output{Bodies: in.Bodies}, nil
	}
	copies, err := m.placeCopies(in.Bodies, []math.Matrix4{math.Identity4(), xf}, "mirror")
	if err != nil {
		return Output{}, err
	}
	if m.def.RemoveOriginal {
		return Output{Bodies: copies}, nil
	}
	if !m.def.JoinToOriginal {
		return Output{Bodies: appendCopies(in.Bodies, copies)}, nil
	}
	joined, err := joinReflections(in.Bodies, copies, in)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: joined}, nil
}

// joinReflections unions each body with its own reflection, so a mirrored half becomes one solid
// with the original rather than two solids that happen to touch. placeCopies emits the reflections
// in body order, so copies[i] is the reflection of bodies[i].
func joinReflections(bodies, copies []*topo.Body, in Input) ([]*topo.Body, error) {
	if len(copies) != len(bodies) {
		return nil, fmt.Errorf("mirror: %d reflections for %d bodies", len(copies), len(bodies))
	}
	out := make([]*topo.Body, 0, len(bodies))
	for i, b := range bodies {
		res, err := ops.BooleanWithDiagnostics(ops.Join, b, copies[i], in.Diag)
		if err != nil {
			return nil, fmt.Errorf("mirror: joining body %d to its reflection: %w", i, err)
		}
		if res == nil || len(res.Faces()) == 0 {
			// The halves do not meet, so there is nothing to union — keep them as two solids
			// rather than dropping one.
			out = append(out, b, copies[i])
			continue
		}
		out = append(out, res)
	}
	return out, nil
}

// ValidateMirrorMode refuses the two body-only options in feature mode instead of ignoring them,
// so a caller that asks for a handed part does not silently get a symmetric one. Exported because
// the op layer checks it when the request is parsed, which names the offending field at the point
// the caller wrote it rather than at the next recompute.
func ValidateMirrorMode(ofBody, removeOriginal, joinToOriginal bool) error {
	if ofBody {
		return nil
	}
	if removeOriginal {
		return fmt.Errorf("mirror: removeOriginal applies to a body mirror only — " +
			"in feature mode there is no original half to discard; set mode to \"body\"")
	}
	if joinToOriginal {
		return fmt.Errorf("mirror: operation applies to a body mirror only — " +
			"a feature mirror already re-applies each source feature's own operation; set mode to \"body\"")
	}
	return nil
}
