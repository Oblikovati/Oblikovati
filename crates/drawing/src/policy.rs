// SPDX-License-Identifier: GPL-2.0-only
//! Modelling and display bounds shared by the desktop and headless hosts.
use oblikovati_martensite::{PrismMeshOptions, PrismResolution, PrismResolutionPolicy};

/// Default display mesh, in database units and radians; example: `let mesh = DISPLAY_MESH`.
pub const DISPLAY_MESH: PrismMeshOptions = PrismMeshOptions {
    chord_error: 0.05,
    angular_error: 0.5,
};

/// Apply the Go ADR-0042 model-relative bounds; example: `modelling_policy().validate()?`.
pub fn modelling_policy() -> PrismResolutionPolicy {
    // Go kernel/geom/resolution.go: a 1 cm reference has weld 1e-9 and sew 1e-4 cm.
    const WELD: f64 = 1e-9; // tol:relative — Go ADR-0042, fraction of profile diagonal.
    const SEW: f64 = 1e-4; // tol:relative — Go ADR-0042, maximum permitted seam gap.
    const ANGULAR: f64 = 1e-9; // tol:angular — Go geom coincidentNormalSinTol.
    const MINIMUM_SIZE: f64 = 1e-9; // tol:relative-floor — Go ADR-0061, in database cm.
    PrismResolutionPolicy::ModelRelative {
        reference: PrismResolution {
            version: 1,
            weld: WELD,
            sew: SEW,
            angular: ANGULAR,
            display_chordal: DISPLAY_MESH.chord_error,
            display_angular: DISPLAY_MESH.angular_error,
            // A result may not carry more uncertainty than the admitted seam gap.
            maximum_tolerance: SEW,
        },
        reference_size: 1.,
        minimum_size: MINIMUM_SIZE,
    }
}
