// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "github.com/Oblikovati/api/contract"

// Compile-time assertion that a planar Sketch satisfies the public scalar contract
// (api/contract.Sketch). The richer entity/constraint access travels as wire DTOs via
// addin/router, not through this interface (ADR-0018, M21-F01).
var _ contract.Sketch = (*Sketch)(nil)

// A non-planar Sketch3D satisfies the public scalar contract (api/contract.Sketch3D),
// the same way the 2D Sketch does (M22-F01).
var _ contract.Sketch3D = (*Sketch3D)(nil)

// A realized sketch Profile satisfies the public Profile contract (area + closed state).
var _ contract.Profile = (*Profile)(nil)
