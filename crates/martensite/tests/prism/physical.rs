// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::{AnalyticBodyProperties, retain_planar_prism};
fn metric(actual: f64, expected: f64) {
    assert!(
        (actual - expected).abs() <= expected.abs().max(1.) * 1e-10,
        "actual {actual}, expected {expected}"
    );
}
#[test]
fn rotated_translated_prism_has_independently_verified_centroidal_inertia() {
    let mut definition = rectangle();
    for vertex in polygon_mut(&mut definition.outer).1 {
        let [x, y] = *vertex;
        *vertex = [0.6 * x - 0.8 * y + 10., 0.8 * x + 0.6 * y - 20.];
    }
    let body = retain_planar_prism(&definition).unwrap();
    let result = body.physical_properties().unwrap();
    metric(result.geometry.volume, 24.);
    metric(result.area, 52.);
    // Exact spectrum certified with Jacobian; R diag(26,40,50) Rᵀ for R=(3/5,4/5).
    let expected = [
        [874. / 25., -168. / 25., 0.],
        [-168. / 25., 776. / 25., 0.],
        [0., 0., 50.],
    ];
    for (actual, expected) in result
        .inertia
        .iter()
        .flatten()
        .zip(expected.iter().flatten())
    {
        metric(*actual, *expected);
    }
    for (actual, expected) in result.principal_moments.into_iter().zip([26., 40., 50.]) {
        metric(actual, expected);
    }
    verify_axes(&result);
    assert_eq!(result, body.physical_properties().unwrap());
}
fn verify_axes(properties: &AnalyticBodyProperties) {
    for i in 0..3 {
        for j in 0..3 {
            let dot: f64 = properties.principal_axes[i]
                .iter()
                .zip(properties.principal_axes[j])
                .map(|(a, b)| a * b)
                .sum();
            metric(dot, if i == j { 1. } else { 0. });
        }
    }
}
#[test]
fn thin_solids_and_holes_keep_positive_inertia_without_display_approximations() {
    for mut definition in [rectangle(), with_hole()] {
        for depth in [2., -2., 1e-5] {
            definition.displacement = [0., 0., depth];
            let result = retain_planar_prism(&definition)
                .unwrap()
                .physical_properties()
                .unwrap();
            assert!(result.principal_moments.iter().all(|value| *value > 0.));
            verify_axes(&result);
        }
    }
}
#[test]
fn analytic_mass_integrals_match_occt_for_every_existing_prism_corpus_shape() {
    let source = include_str!("occt-8.0.0-mass.csv");
    assert_eq!(source.lines().count(), 27);
    for line in source.lines() {
        let row: Vec<f64> = line
            .split(',')
            .map(|value| value.parse().unwrap())
            .collect();
        assert_eq!(row.len(), 19);
        let mut source_row = [0.; 16];
        source_row[..5].copy_from_slice(&row[..5]);
        let definition = super::oracle::oracle_definition(&source_row);
        let result = retain_planar_prism(&definition)
            .unwrap()
            .physical_properties()
            .unwrap();
        for (actual, expected) in flattened_integrals(&result).into_iter().zip(&row[5..]) {
            assert!(
                (actual - expected).abs() <= row[5].abs().max(expected.abs()) * 1e-9,
                "actual {actual}, expected {expected}, case {line}"
            );
        }
        verify_axes(&result);
    }
}
fn flattened_integrals(result: &AnalyticBodyProperties) -> Vec<f64> {
    let mut values = vec![result.geometry.volume, result.area];
    values.extend(result.geometry.centroid);
    for (a, b) in [(0, 0), (1, 1), (2, 2), (0, 1), (1, 2), (2, 0)] {
        values.push(result.inertia[a][b]);
    }
    values.extend(result.principal_moments);
    values
}
