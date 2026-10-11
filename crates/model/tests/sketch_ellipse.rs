// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::sketch::{SketchEllipse2D, SketchEntityId};
fn ellipse(axis: [f64; 2], radii: [f64; 2]) -> SketchEllipse2D {
    SketchEllipse2D::new(SketchEntityId(2), SketchEntityId(1), axis, radii).unwrap()
}
#[test]
fn occt_position_and_derivative_match_across_scales_frames_and_thin_ellipses() {
    let fixture = include_str!("sketch_ellipse/occt-8.0.0.csv");
    assert_eq!(fixture.lines().count(), 108);
    for row in fixture.lines() {
        assert_occt_sample(row);
    }
}
fn assert_occt_sample(row: &str) {
    let sample: Vec<f64> = row
        .split(',')
        .map(|value| value.parse::<f64>().unwrap())
        .collect();
    let [cx, cy, ax, ay, major, minor, angle, px, py, tx, ty]: [f64; 11] =
        sample.try_into().unwrap();
    let value = ellipse([ax, ay], [major, minor]);
    let actual = value.evaluate([cx, cy], angle).unwrap();
    let tolerance = 32.0 * f64::EPSILON * major.max(cx.abs()).max(cy.abs());
    assert_oracle_components(actual, [px, py, tx, ty], tolerance, row);
}
fn assert_oracle_components(
    actual: oblikovati_model::sketch::SketchEllipseEvaluation,
    expected: [f64; 4],
    tolerance: f64,
    row: &str,
) {
    for (actual, expected) in actual
        .position
        .into_iter()
        .chain(actual.tangent)
        .zip(expected)
    {
        assert!(
            (actual - expected).abs() <= tolerance,
            "{row}: {actual} expected {expected}"
        );
    }
}

#[test]
fn derivatives_agree_with_an_independent_symmetric_difference() {
    let value = ellipse([3.0, 4.0], [3.0, 1.0]);
    let step = 1e-5;
    for parameter in [0.13, 1.17, -2.5] {
        let actual = value.evaluate([7.0, -5.0], parameter).unwrap();
        let before = value.evaluate([7.0, -5.0], parameter - step).unwrap();
        let after = value.evaluate([7.0, -5.0], parameter + step).unwrap();
        for axis in 0..2 {
            let derivative = (after.position[axis] - before.position[axis]) / (2.0 * step);
            assert!((derivative - actual.tangent[axis]).abs() < 1e-9);
        }
    }
}
#[test]
fn identity_and_metric_domain_errors_are_named_and_never_repaired() {
    for (id, center) in [(0, 1), (1, 0), (1, 1)] {
        let error = SketchEllipse2D::new(
            SketchEntityId(id),
            SketchEntityId(center),
            [1.0, 0.0],
            [3.0, 1.0],
        )
        .unwrap_err();
        assert!(error.to_string().contains("distinct nonzero"));
    }
}
#[test]
fn unordered_or_nonpositive_radii_cannot_construct_or_edit_an_ellipse() {
    for radii in [
        [0.0, 0.0],
        [1.0, 2.0],
        [3.0, -1.0],
        [f64::INFINITY, 1.0],
        [3.0, f64::NAN],
    ] {
        let error = SketchEllipse2D::new(SketchEntityId(2), SketchEntityId(1), [1.0, 0.0], radii)
            .unwrap_err();
        assert!(error.to_string().contains("major >= minor > 0"));
        assert!(ellipse([1.0, 0.0], [3.0, 1.0]).with_radii(radii).is_err());
    }
}
#[test]
fn invalid_axes_angles_and_nonrepresentable_positions_decline_explicitly() {
    for axis in [[0.0, 0.0], [f64::NAN, 0.0], [f64::INFINITY, 1.0]] {
        assert!(
            SketchEllipse2D::new(SketchEntityId(2), SketchEntityId(1), axis, [3.0, 1.0])
                .unwrap_err()
                .to_string()
                .contains("major axis")
        );
    }
}
#[test]
fn invalid_angles_and_nonrepresentable_positions_decline_explicitly() {
    let value = ellipse([1.0, 0.0], [3.0, 1.0]);
    for angle in [f64::NAN, f64::INFINITY, f64::NEG_INFINITY] {
        assert!(value.with_orientation(angle).is_err());
        assert!(value.evaluate([0.0, 0.0], angle).is_err());
    }
    assert!(value.evaluate([f64::NAN, 0.0], 0.0).is_err());
    let large = ellipse([1.0, 0.0], [f64::MAX, 1.0]);
    assert!(
        large
            .evaluate([f64::MAX, 0.0], 0.0)
            .unwrap_err()
            .to_string()
            .contains("finite analytic")
    );
}
#[test]
fn unchanged_orientation_preserves_axis_bits_and_rotation_has_no_magnitude_freedom() {
    let value = ellipse([3.0, 4.0], [3.0, 1.0]);
    assert_eq!(value.id(), SketchEntityId(2));
    assert_eq!(value.center(), SketchEntityId(1));
    assert_eq!(value.radii(), [3.0, 1.0]);
    assert_eq!(value.major_axis(), [0.6, 0.8]);
    let unchanged = value.with_orientation(value.orientation()).unwrap();
    for (first, second) in value.major_axis().into_iter().zip(unchanged.major_axis()) {
        assert_eq!(first.to_bits(), second.to_bits());
    }
    let rotated = value.with_orientation(std::f64::consts::FRAC_PI_2).unwrap();
    assert_eq!(rotated.radii(), value.radii());
    assert!((rotated.major_axis()[1] - 1.0).abs() < f64::EPSILON);
    assert_eq!(
        value.with_radii([4.0, 1.0]).unwrap().major_axis(),
        value.major_axis()
    );
}
#[test]
fn positive_subnormal_axes_and_equal_radii_remain_valid_analytic_values() {
    let value = ellipse([f64::from_bits(1), 0.0], [1.0, 1.0]);
    assert_eq!(value.major_axis(), [1.0, 0.0]);
    assert_eq!(
        value.evaluate([0.0, 0.0], 0.0).unwrap().position,
        [1.0, 0.0]
    );
}

#[test]
fn axis_normalization_preserves_directions_at_finite_extremes() {
    for magnitude in [f64::from_bits(1), f64::MIN_POSITIVE, 1.0, f64::MAX] {
        let value = ellipse([magnitude, magnitude], [3.0, 1.0]);
        let [x, y] = value.major_axis();
        assert_eq!(x, y);
        assert!((x.hypot(y) - 1.0).abs() < 2.0 * f64::EPSILON);
        assert_eq!(value.orientation(), std::f64::consts::FRAC_PI_4);
    }
}
