// SPDX-License-Identifier: GPL-2.0-only
//! Versioned desktop defaults derived from the Go modelling and display policies.
use oblikovati_martensite::{PrismResolution, PrismResolutionPolicy};

/// Apply the Go ADR-0042 model-relative weld and sew bounds to each solved profile.
/// Example: `owner.with_extrusion_policy(desktop_policy())?`.
pub(crate) fn desktop_policy() -> PrismResolutionPolicy {
    // Go kernel/geom/resolution.go: a 1 cm reference has weld 1e-9 and sew 1e-4 cm.
    const WELD: f64 = 1e-9; // tol:relative — Go ADR-0042, fraction of profile diagonal.
    const SEW: f64 = 1e-4; // tol:relative — Go ADR-0042, maximum permitted seam gap.
    const ANGULAR: f64 = 1e-9; // tol:angular — Go geom coincidentNormalSinTol.
    const MINIMUM_SIZE: f64 = 1e-9; // tol:relative-floor — Go ADR-0061, in database cm.
    let display = super::DISPLAY_MESH;
    PrismResolutionPolicy::ModelRelative {
        reference: PrismResolution {
            version: 1,
            weld: WELD,
            sew: SEW,
            angular: ANGULAR,
            display_chordal: display.chord_error,
            display_angular: display.angular_error,
            // A result may not carry more uncertainty than the admitted seam gap.
            maximum_tolerance: SEW,
        },
        reference_size: 1.,
        minimum_size: MINIMUM_SIZE,
    }
}
