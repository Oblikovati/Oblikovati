# Geometric constraint creation

Tracking: [#3781](https://github.com/Oblikovati/Oblikovati/issues/3781).
Deletion is recorded in [the deletion evidence](implementation-constraint-deletion.md).

The host consumes the released API's `AddConstraintArgs`, `AddConstraintResult`,
method constant, and typed `client.constrain().add(&args)` or `add_with` call.
Constructor selection uses the canonical kind and actual owned entity kinds.
It selects one constructor; it does not try several geometry operations.

All 18 current geometric model forms are available: coincidence, point alignment,
single-line horizontal/vertical, point fix, line/arc midpoint, point-on-line,
point-on-circle/arc, four line relation forms, concentric curves, equal radius,
and line/circular or circular/circular tangency. Mixed operands use the Go
constructor order. Unsupported Go forms, operand counts, kinds, or owners receive
named declines before allocation. Ellipse flags do not change point or line
constructors. The Go built-in constructor behavior also ignores custom metadata;
no unsupported custom relation or name is fabricated.

For example, a horizontal request with one line returns `horizontal`; the same
request with two points returns `horizontalAlign`. The result kind describes the
created relation, and its index addresses the canonical geometric list. Dimensions
and structural equations are excluded from that index.

The model validates owned references, reserves one permanent sketch-local ID,
appends on isolated geometry, and evaluates an isolated document candidate with
current parameter and projection inputs. It adopts that candidate once. The
returned DOF is the resulting numerical structure. Fix and tangent use current
coordinates before capturing their seed. Tangent side and containment selection
remain in the existing model constructors. The source-coordinate preparation
previously used by inferred lines is now shared with these seed-dependent edits;
its existing convergence requirement and free-variable support are preserved.

The normal session command prepares the response before publication and retains
automatic recompute for other pending readers. A numerical nonconvergence retains
explicit sketch health. A compile or contact-selection failure publishes no
geometry and retains attempted reservations. Response rejection, commit veto,
group abort, deletion, undo, and storage reload cannot reuse those identities.

Tests cover the complete constructor vocabulary and actual returned kinds in
part and assembly documents, circle/arc permutations, invalid mixed orders,
unsupported forms, malformed requests, wrong owners, current driven coordinates,
mixed dimensional indices, degenerate and near-degenerate lines, rotated/scaled
and reversed inputs, numerical health, exhaustion, caught failure, response and
commit rejection, transaction groups, undo/redo, save/reopen, and copy. The pinned
Go router tests supply independent constructor and deletion reference evidence.

The full local gate covers every Rust crate present in the application workspace.
Native application rendering and released kernel acceptance remain open.
