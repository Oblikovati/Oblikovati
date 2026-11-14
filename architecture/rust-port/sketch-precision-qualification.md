# Sketch precision qualification

Issue #3928 tracks qualification of solved sketches for profile extrusion.
The current implementation retains raw residual rows, source identities, convergence
scaling domains, and conservative physical-unit labels. These labels are necessary
inputs for qualification; they are not numerical error bounds and are not used to
admit extrusion. Unknown or mixed rows remain `Unknown`.

A qualification implementation must use the current solved sketch and its current
projection inputs. For each relevant constraint, it must retain source identities,
identify operand provenance and physical units, and determine whether conditioning
permits a conservative spatial or angular error bound. A local Jacobian or a raw
residual alone is not such a bound. Angular quantities need a certified geometric
conversion before comparison with a length tolerance. Degenerate, ambiguous,
unsupported, stale, failed, and unconverged inputs must produce named declines.

The qualified bounds must be checked using the tolerance class appropriate to each
operand's origin. The same evidence and policy must reach profile discovery,
preview, creation, edit, and ordered replay. Result validation, identity and
provenance checks remain required. No geometry may be moved to satisfy a bound.

A future gate must cover known and unknown constraint domains, short and long
operands, ill-conditioned and degenerate inputs, free and fully constrained
sketches, mixed-unit rows, projected inputs, edits and replay. Independent analytic
oracles must check the reported bounds. Only after that gate passes can the general
zero-DOF restriction be reconsidered for #3927.
