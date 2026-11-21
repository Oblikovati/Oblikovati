# Sketch precision qualification investigation

Issue: [#3928](https://github.com/Oblikovati/Oblikovati/issues/3928).
The application does not yet have a certified spatial or angular solve bound.
Convergence, zero DOF, and a small residual do not establish such a bound;
the solver and point-on-line regression tests retain counterexamples.

An isolated interval-contraction experiment passed nine tests, formatting,
and Clippy. It accepted only conditional enclosures for square systems when
its supplied function and Jacobian intervals met its inclusion and contraction
checks. This was not connected to application sketches or independently audited;
it is not a precision certificate and did not change production behavior.

The next implementation must provide:

- Outward-rounded evaluations and Jacobian enclosures for the actual supported
  constraint equations, with domain and branch checks. Point evaluations and
  mixed residual norms are insufficient.
- Bounded, source-attributed input uncertainty for parameter values, fixed
  coordinates, and projected anchors.
- A justified treatment of free variables and overdetermined equations. The
  square-system experiment does not establish either case.
- Separate geometric spatial and angular bounds, including conditioning and
  degeneracy checks, without moving geometry or changing tolerances silently.
- One qualification decision shared by profile discovery, preview, creation,
  edit, and replay, including direct sketch-prism entry points. Unsupported or
  inconclusive cases must produce a named refusal.

The prototype assumed stable IEEE-754 arithmetic with gradual underflow. Its
local subnormal tests did not establish that assumption across platforms.
