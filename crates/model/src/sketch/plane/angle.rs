// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl SketchPlane {
    /// Rotate a base normal about a checked axis; example: `base.at_angle_about_axis([0.0;3], [1.0,0.0,0.0], std::f64::consts::FRAC_PI_2)?`.
    pub fn at_angle_about_axis(
        self,
        origin: [f64; 3],
        direction: [f64; 3],
        radians: f64,
    ) -> Result<Self, SketchError> {
        require_unit(direction, self.options.unit_deviation)?;
        require_finite(&[radians], "line-plane angle in radians")?;
        let projection = dot_product(self.normal, direction);
        let reference = normalize_direction(std::array::from_fn(|i| {
            self.normal[i] - direction[i] * projection
        }))?;
        // #3758: projection roundoff must not manufacture roll when the base normal follows the axis.
        require_perpendicular(direction, reference, self.options.angular_deviation)?;
        let rotated = normalize_direction(rotate_normal(reference, direction, radians))?;
        require_perpendicular(direction, rotated, self.options.angular_deviation)?;
        Self::from_origin_normal(origin, rotated, self.options)
    }
}
fn rotate_normal(normal: [f64; 3], direction: [f64; 3], radians: f64) -> [f64; 3] {
    // #3758: preserve the Go aa8297d1 Rodrigues matrix order; vector rotation has no translation.
    let c = radians.cos();
    let s = radians.sin();
    let t = 1.0 - c;
    let [x, y, z] = direction;
    let rows = [
        [t * x * x + c, t * x * y - s * z, t * x * z + s * y],
        [t * x * y + s * z, t * y * y + c, t * y * z - s * x],
        [t * x * z - s * y, t * y * z + s * x, t * z * z + c],
    ];
    rows.map(|row| dot_product(row, normal))
}
