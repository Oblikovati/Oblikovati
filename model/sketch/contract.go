// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati/api/contract"

// Compile-time assertion that a planar Sketch satisfies the public scalar contract
// (api/contract.Sketch). The richer entity/constraint access travels as wire DTOs via
// addin/router, not through this interface (ADR-0018, M21-F01).
var _ contract.Sketch = (*Sketch)(nil)

// A non-planar Sketch3D satisfies the public scalar contract (api/contract.Sketch3D),
// the same way the 2D Sketch does (M22-F01).
var _ contract.Sketch3D = (*Sketch3D)(nil)

// A realized sketch Profile satisfies the public Profile contract (area + closed state).
var _ contract.Profile = (*Profile)(nil)

// A closed planar 3D-sketch loop satisfies the public Profile3D contract (M22-F09).
var _ contract.Profile3D = (*Profile3D)(nil)
