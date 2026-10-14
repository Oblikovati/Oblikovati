// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_martensite::{PlanarSplineDefinition, evaluate_planar_spline};
#[path = "planar_spline/go_recipe_fixture.rs"]
mod fixture;
use fixture::{FIT_POINTS, GoSplineOracle};

#[test]
fn published_kernel_evaluates_all_go_fit_methods_without_changing_the_recipe() {
    let records: Vec<GoSplineOracle> =
        serde_json::from_str(include_str!("planar_spline/go-fit-oracle.json")).unwrap();
    assert_eq!(records.len(), 6);
    for (index, record) in records.iter().enumerate() {
        assert_eq!(record.method, (index % 3) as u8);
        assert_eq!(record.closed, index >= 3);
        assert_go_recipe(record);
    }
}

fn assert_go_recipe(record: &GoSplineOracle) {
    let curve = native_definition(record);
    let before = curve.clone();
    for &[parameter, px, py, tx, ty] in &record.samples {
        let sample = evaluate_planar_spline(&curve, parameter, 0.0).unwrap();
        assert_coordinates(
            sample.position.into_iter().chain(sample.tangent),
            [px, py, tx, ty],
        );
    }
    assert_eq!(curve, before);
    assert_fit_stations(record, &curve);
}

fn native_definition(record: &GoSplineOracle) -> PlanarSplineDefinition {
    let (knots, multiplicities) = distinct_knots(&record.knots);
    // Go stores closure as wrapped poles and extended knots, not a periodic flag.
    PlanarSplineDefinition {
        poles: record.poles.clone(),
        weights: None,
        knots,
        multiplicities,
        degree: record.degree,
        periodic: false,
    }
}

fn distinct_knots(flat: &[f64]) -> (Vec<f64>, Vec<i32>) {
    let mut knots: Vec<f64> = Vec::new();
    let mut multiplicities: Vec<i32> = Vec::new();
    for &knot in flat {
        if knots.last() == Some(&knot) {
            *multiplicities.last_mut().unwrap() += 1;
            continue;
        }
        knots.push(knot);
        multiplicities.push(1);
    }
    (knots, multiplicities)
}

fn assert_fit_stations(record: &GoSplineOracle, curve: &PlanarSplineDefinition) {
    let points = FIT_POINTS;
    assert_eq!(
        record.stations.len(),
        points.len() + usize::from(record.closed)
    );
    for (index, &parameter) in record.stations.iter().enumerate() {
        let sample = evaluate_planar_spline(curve, parameter, 0.0).unwrap();
        assert_coordinates(sample.position.into_iter(), points[index % points.len()]);
    }
    if record.closed {
        let first = evaluate_planar_spline(curve, 0.0, 0.0).unwrap();
        let last = evaluate_planar_spline(curve, 1.0, 0.0).unwrap();
        assert_coordinates(first.tangent.into_iter(), last.tangent);
    }
}

fn assert_coordinates<const N: usize>(actual: impl Iterator<Item = f64>, expected: [f64; N]) {
    for (actual, expected) in actual.zip(expected) {
        let bound = 256.0 * f64::EPSILON * expected.abs().max(1.0);
        assert!(
            (actual - expected).abs() <= bound,
            "{actual}: expected {expected} within {bound}"
        );
    }
}
