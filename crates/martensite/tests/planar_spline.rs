// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_martensite::{PlanarSplineDefinition, PlanarSplineSample, evaluate_planar_spline};
fn definition(kind: u8, scale: f64) -> PlanarSplineDefinition {
    let mut poles = vec![[0.0, 0.0], [scale, 2.0 * scale], [3.0 * scale, 0.0]];
    if kind == 2 {
        poles.push([scale, -scale]);
    }
    PlanarSplineDefinition {
        poles,
        weights: (kind == 1).then(|| vec![1.0, 0.5, 1.0]),
        knots: if kind == 2 {
            vec![0.0, 1.0, 2.0, 3.0, 4.0]
        } else {
            vec![0.0, 1.0]
        },
        multiplicities: if kind == 2 { vec![1; 5] } else { vec![3; 2] },
        degree: 2,
        periodic: kind == 2,
    }
}
#[test]
fn published_kernel_parent_and_offset_match_135_occt_analytic_samples() {
    let fixture = include_str!("planar_spline/occt-8.0.0.csv");
    assert_eq!(fixture.lines().count(), 135);
    for row in fixture.lines() {
        assert_oracle_row(row);
    }
}
fn assert_oracle_row(row: &str) {
    let values: Vec<f64> = row.split(',').map(|value| value.parse().unwrap()).collect();
    let [kind, scale, left, parameter, px, py, tx, ty]: [f64; 8] = values.try_into().unwrap();
    let curve = definition(kind as u8, scale);
    let before = curve.clone();
    let sample = evaluate_planar_spline(&curve, parameter, left).unwrap();
    for (actual, expected) in sample
        .position
        .into_iter()
        .chain(sample.tangent)
        .zip([px, py, tx, ty])
    {
        assert!(
            (actual - expected).abs() <= 128.0 * f64::EPSILON * scale,
            "{row}: {actual} expected {expected}"
        );
    }
    assert_eq!(curve, before);
}
#[test]
fn signed_left_offset_uses_exact_parent_normal_and_preserves_parameter_direction() {
    for kind in [0, 1, 2] {
        for left in [-0.25, 0.25] {
            assert_signed_normal(kind, left);
        }
    }
}
fn assert_signed_normal(kind: u8, left: f64) {
    let curve = definition(kind, 1.0);
    let basis = evaluate_planar_spline(&curve, 0.5, 0.0).unwrap();
    let offset = evaluate_planar_spline(&curve, 0.5, left).unwrap();
    let length = basis.tangent[0].hypot(basis.tangent[1]);
    let delta = [
        offset.position[0] - basis.position[0],
        offset.position[1] - basis.position[1],
    ];
    assert!((delta[0] + left * basis.tangent[1] / length).abs() < 1e-14);
    assert!((delta[1] - left * basis.tangent[0] / length).abs() < 1e-14);
    assert!(basis.tangent[0] * offset.tangent[0] + basis.tangent[1] * offset.tangent[1] > 0.0);
    assert_independent_tangent(&curve, left, offset);
}
fn assert_independent_tangent(
    curve: &PlanarSplineDefinition,
    left: f64,
    sample: PlanarSplineSample,
) {
    let step = 1e-5;
    let before = evaluate_planar_spline(curve, 0.5 - step, left).unwrap();
    let after = evaluate_planar_spline(curve, 0.5 + step, left).unwrap();
    for axis in 0..2 {
        let derivative = (after.position[axis] - before.position[axis]) / (2.0 * step);
        assert!((derivative - sample.tangent[axis]).abs() < 1e-8);
    }
}
#[test]
fn invalid_domains_and_nonfinite_values_are_named_refusals() {
    for (parameter, left) in [
        (f64::NAN, 0.0),
        (0.5, f64::INFINITY),
        (-0.1, 0.0),
        (1.1, 0.0),
    ] {
        let error = evaluate_planar_spline(&definition(0, 1.0), parameter, left).unwrap_err();
        assert!(error.contains("expected"));
        assert!(error.contains(&parameter.to_string()));
    }
    for slot in [0, 1, 2] {
        assert_bad_definition(slot);
    }
}
fn assert_bad_definition(slot: u8) {
    let mut curve = definition(1, 1.0);
    match slot {
        0 => curve.poles[0][0] = f64::INFINITY,
        1 => curve.knots[0] = f64::NAN,
        _ => curve.weights.as_mut().unwrap()[0] = f64::INFINITY,
    }
    assert!(
        evaluate_planar_spline(&curve, 0.5, 0.0)
            .unwrap_err()
            .contains("expected finite")
    );
}
#[test]
fn malformed_native_representations_do_not_become_polygons() {
    for slot in 0..5 {
        assert_native_decline(slot);
    }
}
fn assert_native_decline(slot: u8) {
    let mut curve = definition(0, 1.0);
    match slot {
        0 => curve.degree = 0,
        1 => curve.multiplicities = vec![1, 1],
        2 => curve.knots = vec![1.0, 0.0],
        3 => curve.weights = Some(vec![1.0]),
        _ => curve.weights = Some(vec![1.0, -1.0, 1.0]),
    }
    assert!(
        evaluate_planar_spline(&curve, 0.5, 0.0)
            .unwrap_err()
            .contains("expected a valid analytic B-spline")
    );
}
#[test]
fn stationary_parent_is_refused_for_nonzero_offset_without_perturbation() {
    let mut curve = definition(0, 1.0);
    curve.poles.fill([0.0, 0.0]);
    let before = curve.clone();
    let sample = evaluate_planar_spline(&curve, 0.5, 0.0).unwrap();
    assert_eq!(sample.tangent, [0.0, 0.0]);
    let error = evaluate_planar_spline(&curve, 0.5, 1.0).unwrap_err();
    assert!(error.contains("nonstationary"));
    assert_eq!(curve, before);
}

#[test]
fn exact_offset_cusp_returns_a_named_decline() {
    let mut curve = definition(0, 1.0);
    curve.poles = vec![[-1.0, 1.0], [0.0, -1.0], [1.0, 1.0]];
    let before = curve.clone();
    let error = evaluate_planar_spline(&curve, 0.5, 0.5).unwrap_err();
    assert!(error.contains("nonstationary"));
    assert_eq!(curve, before);
}
#[test]
fn nonrepresentable_derivative_is_refused_before_returning_a_sample() {
    let mut curve = definition(0, 1.0);
    curve.poles = vec![[-1e308, 0.0], [0.0, 0.0], [1e308, 0.0]];
    let error = evaluate_planar_spline(&curve, 0.5, 0.0).unwrap_err();
    assert!(error.contains("expected finite"));
}
