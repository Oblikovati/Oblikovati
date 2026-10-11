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
        frame_from_axis_and_displacement(first, x, toward_third, options)
    }
    /// Use ordered checked axis directions; example: `SketchPlane::from_axis_directions([0.0;3], [1.0,0.0,0.0], [0.0,1.0,0.0], Default::default())?`.
    pub fn from_axis_directions(
        origin: [f64; 3],
        first: [f64; 3],
        second: [f64; 3],
        options: SketchPlaneOptions,
    ) -> Result<Self, SketchError> {
        require_options(options)?;
        require_unit(first, options.unit_deviation)?;
        require_unit(second, options.unit_deviation)?;
        // #3757: Go aa8297d1 defines roll from directions; the second origin has no placement role.
        frame_from_axis_and_displacement(origin, first, second, options)
    }
    /// Contain a checked unit line and one off-line point; example: `SketchPlane::through_line_and_point([0.0;3], [1.0,0.0,0.0], [0.0,1.0,0.0], Default::default())?`.
    pub fn through_line_and_point(
        origin: [f64; 3],
        direction: [f64; 3],
        point: [f64; 3],
        options: SketchPlaneOptions,
    ) -> Result<Self, SketchError> {
        require_options(options)?;
        require_finite(&origin, "line origin")?;
        require_finite(&point, "through point")?;
        require_unit(direction, options.unit_deviation)?;
        let toward = normalize_direction(point_difference(origin, point))?;
        frame_from_axis_and_displacement(origin, direction, toward, options)
    }
}

fn frame_from_axis_and_displacement(
    origin: [f64; 3],
    x: [f64; 3],
    toward: [f64; 3],
    options: SketchPlaneOptions,
) -> Result<SketchPlane, SketchError> {
    let normal = normalize_direction(cross_product(x, toward))?;
    let y = normalize_direction(cross_product(normal, x))?;
    SketchPlane::new(origin, x, y, options)
}
