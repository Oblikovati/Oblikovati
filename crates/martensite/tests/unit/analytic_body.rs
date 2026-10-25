// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn invalid_integrals_and_principal_certificates_are_refused() {
    let shape = crate::build_probe([4., 3., 2.]).unwrap();
    let valid = integrate_body(&shape).unwrap();
    let mut zero_volume = valid.clone();
    zero_volume.geometry.volume = 0.;
    assert!(certify_integrals(&zero_volume).is_err());
    let mut nonfinite = valid.clone();
    nonfinite.area = f64::INFINITY;
    assert!(certify_integrals(&nonfinite).is_err());
    let mut bad_tensor = valid.clone();
    bad_tensor.inertia[0][0] = f64::NAN;
    assert!(certify_integrals(&bad_tensor).is_err());
    assert!(certify_principal(valid.inertia, 26., [2., 0., 0.]).is_err());
    assert!(certify_principal(valid.inertia, 10., [1., 0., 0.]).is_err());
    assert!(certify_principal(valid.inertia, -1., [1., 0., 0.]).is_err());
    assert!(certify_orthogonal([[1., 0., 0.]; 3]).is_err());
}
#[test]
fn canonical_axis_uses_stable_sign_ties_and_positive_zero() {
    assert_eq!(canonical_axis([-0.6, -0.8, 0.]), [0.6, 0.8, 0.]);
    assert_eq!(canonical_axis([-1., 1., -0.]), [1., -1., 0.]);
    assert!(!canonical_axis([1., 0., -0.])[2].is_sign_negative());
}
fn offset_box(origin: [f64; 3], sizes: [f64; 3]) -> Shape {
    use martensite_tkmath::gp::Point3;
    use martensite_tkprim::brep_prim_api::BRepPrimAPIMakeBox;
    BRepPrimAPIMakeBox::at_point(
        Point3::new(origin[0], origin[1], origin[2]),
        sizes[0],
        sizes[1],
        sizes[2],
    )
    .unwrap()
    .solid()
    .unwrap()
    .into()
}
#[test]
fn system_preserves_single_body_bits_and_empty_has_no_fabricated_solid() {
    assert!(integrate_system([]).unwrap().is_none());
    let shape = offset_box([10., -20., 30.], [4., 3., 2.]);
    assert_eq!(
        integrate_body(&shape).unwrap(),
        integrate_system([&shape]).unwrap().unwrap()
    );
}
#[test]
fn separated_boxes_include_parallel_axis_inertia_about_the_combined_centroid() {
    let first = offset_box([0., 0., 0.], [4., 3., 2.]);
    let second = offset_box([10., 0., 0.], [4., 3., 2.]);
    let result = integrate_system([&first, &second]).unwrap().unwrap();
    near(result.geometry.volume, 48.);
    near(result.area, 104.);
    for (actual, expected) in result.geometry.centroid.into_iter().zip([7., 1.5, 1.]) {
        near(actual, expected);
    }
    for (actual, expected) in result.inertia.iter().flatten().zip(
        [[52., 0., 0.], [0., 1280., 0.], [0., 0., 1300.]]
            .iter()
            .flatten(),
    ) {
        near(*actual, *expected);
    }
    let reverse = integrate_system([&second, &first]).unwrap().unwrap();
    for (actual, expected) in result
        .inertia
        .iter()
        .flatten()
        .zip(reverse.inertia.iter().flatten())
    {
        near(*actual, *expected);
    }
}
fn near(actual: f64, expected: f64) {
    assert!(
        (actual - expected).abs() <= expected.abs().max(1.) * 1e-10,
        "actual {actual}, expected {expected}"
    );
}
#[test]
fn combined_integrals_match_occt_at_three_scales_and_offsets() {
    let source = include_str!("../analytic_system/occt-8.0.0.csv");
    assert_eq!(source.lines().count(), 9);
    for line in source.lines() {
        let row: Vec<f64> = line
            .split(',')
            .map(|value| value.parse().unwrap())
            .collect();
        assert_eq!(row.len(), 18);
        let scale = row[0];
        let first = offset_box([0.; 3], [4. * scale, 3. * scale, 2. * scale]);
        let second = offset_box(
            [row[1] * scale, row[2] * scale, row[3] * scale],
            [4. * scale, 3. * scale, 2. * scale],
        );
        let result = integrate_system([&first, &second]).unwrap().unwrap();
        for (index, (actual, expected)) in system_values(&result).iter().zip(&row[4..]).enumerate()
        {
            let power = match index {
                0 => 3,
                1 => 2,
                2..=4 => 1,
                _ => 5,
            };
            assert!(
                (actual - expected).abs() <= expected.abs().max(scale.powi(power)) * 1e-9,
                "actual {actual}, expected {expected}, case {line}"
            );
        }
    }
}
fn system_values(result: &AnalyticBodyProperties) -> Vec<f64> {
    let mut values = vec![result.geometry.volume, result.area];
    values.extend(result.geometry.centroid);
    for (a, b) in [(0, 0), (1, 1), (2, 2), (0, 1), (1, 2), (2, 0)] {
        values.push(result.inertia[a][b]);
    }
    values.extend(result.principal_moments);
    values
}
#[test]
fn thin_separated_bodies_keep_positive_finite_combined_moments() {
    for thickness in [1e-5, 2e-5, 1e-4] {
        let first = offset_box([0.; 3], [4., 3., thickness]);
        let second = offset_box([1e-6, -1e-6, thickness], [4., 3., thickness]);
        let result = integrate_system([&first, &second]).unwrap().unwrap();
        near(result.geometry.volume, 24. * thickness);
        assert!(
            result
                .principal_moments
                .iter()
                .all(|moment| moment.is_finite() && *moment > 0.)
        );
    }
}
