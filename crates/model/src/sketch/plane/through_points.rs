// SPDX-License-Identifier: GPL-2.0-only
use super::*;

impl SketchPlane {
    /// Derive a right-handed frame from ordered points; example: `SketchPlane::through_points([0.0;3], [1.0,0.0,0.0], [0.0,1.0,0.0], Default::default())?`.
    pub fn through_points(
        first: [f64; 3],
        second: [f64; 3],
        third: [f64; 3],
        options: SketchPlaneOptions,
    ) -> Result<Self, SketchError> {
        for point in [first, second, third] {
            require_finite(&point, "three-point frame driver")?;
        }
        // #3754: normalize displacements before cross products to retain extreme finite scales.
        let x = normalize_direction(point_difference(first, second))?;
        let toward_third = normalize_direction(point_difference(first, third))?;
        let normal = normalize_direction(cross_product(x, toward_third))?;
        let y = normalize_direction(cross_product(normal, x))?;
        Self::new(first, x, y, options)
    }
}
