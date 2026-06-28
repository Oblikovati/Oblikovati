// SPDX-License-Identifier: GPL-2.0-only

package clientgraphics

import "oblikovati.org/api/contract"

// A Group satisfies the public scalar view of a client-graphics group (api/contract.
// ClientGraphics). Richer access (the geometry itself) travels as wire DTOs, decoded by
// DecodeGroup — no GPL model types cross the boundary.
var _ contract.ClientGraphics = (*Group)(nil)
