// SPDX-License-Identifier: GPL-2.0-only

package param

import "oblikovati/api/contract"

// Parameter implements the Apache-2.0 in-process contract. This assertion keeps the
// implementation and the published interface honest at compile time (ADR-0018): if
// a method's signature drifts from [contract.Parameter], the build breaks here
// rather than silently diverging the public surface from what /source actually does.
var _ contract.Parameter = (*Parameter)(nil)
