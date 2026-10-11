// SPDX-License-Identifier: GPL-2.0-only
use super::*;

// #3756: retain Go aa8297d1's conditioned seed choice; this controls frame roll, not modelling tolerance.
const NORMAL_FRAME_X_SEED_MAX_ALIGNMENT: f64 = 0.9;

impl SketchPlane {
    /// Construct a deterministic frame around a checked unit normal; example: `SketchPlane::from_origin_normal([0.0;3], [0.0,0.0,1.0], Default::default())?`.
    pub fn from_origin_normal(
        origin: [f64; 3],
        normal: [f64; 3],
        options: SketchPlaneOptions,
    ) -> Result<Self, SketchError> {
        require_options(options)?;
        require_unit(normal, options.unit_deviation)?;
        // These definitions constrain the normal but leave the in-plane roll free.
        let x = normal_frame_x_axis(normal)?;
        let y = normalize_direction(cross_product(normal, x))?;
        Self::new(origin, x, y, options)
    }
}
fn normal_frame_x_axis(normal: [f64; 3]) -> Result<[f64; 3], SketchError> {
    let seed = if normal[0].abs() > NORMAL_FRAME_X_SEED_MAX_ALIGNMENT {
        [0.0, 1.0, 0.0]
    } else {
        [1.0, 0.0, 0.0]
    };
    normalize_direction(cross_product(normal, seed))
}
