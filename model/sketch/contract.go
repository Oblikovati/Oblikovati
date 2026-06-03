// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "github.com/Oblikovati/api/contract"

// Compile-time assertion that a planar Sketch satisfies the public scalar contract
// (api/contract.Sketch). The richer entity/constraint access travels as wire DTOs via
// addin/router, not through this interface (ADR-0018, M21-F01).
var _ contract.Sketch = (*Sketch)(nil)
