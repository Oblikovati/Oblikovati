# Sealed source namespace probe

On Linux, the experimental sealed runner passed the pinned document-opening
scenario: one scenario and nine steps passed. It checked the complete mounted
source checkout, including Git metadata, and the runtime before and after the
run. The source revision was `8ec8d2c1e569fbc5c8cd2ec85e9b9380c62c147f`.
Nested user namespaces were disabled. The private Xvfb display was closed when
the command ended.

This result is a feasibility probe, not a reusable scenario receipt. The
host can change a mounted input during execution, and host and device inputs
are not fully bounded. The observation continues to reject reuse (#3974).
