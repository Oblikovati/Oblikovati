// SPDX-License-Identifier: GPL-2.0-only
pub(super) fn triangle_normal(points: [[f32; 3]; 3]) -> [f32; 3] {
    let [a, b, c] = points.map(|point| point.map(f64::from));
    let u: [f64; 3] = std::array::from_fn(|axis| b[axis] - a[axis]);
    let v: [f64; 3] = std::array::from_fn(|axis| c[axis] - a[axis]);
    let normal = [
        u[1] * v[2] - u[2] * v[1],
        u[2] * v[0] - u[0] * v[2],
        u[0] * v[1] - u[1] * v[0],
    ];
    let length = normal.iter().map(|part| part * part).sum::<f64>().sqrt();
    if length == 0. {
        return [0.; 3];
    }
    normal.map(|part| (part / length) as f32)
}

#[cfg(test)]
mod tests {
    use super::triangle_normal;

    #[test]
    fn normal_respects_orientation_and_finite_extreme_coordinates() {
        for extent in [f32::MIN_POSITIVE, 1., f32::MAX] {
            let points = [[0., 0., 0.], [extent, 0., 0.], [0., extent, 0.]];
            assert_eq!(triangle_normal(points), [0., 0., 1.]);
            assert_eq!(
                triangle_normal([points[0], points[2], points[1]]),
                [0., 0., -1.]
            );
        }
        assert_eq!(triangle_normal([[0.; 3]; 3]), [0.; 3]);
    }
}
