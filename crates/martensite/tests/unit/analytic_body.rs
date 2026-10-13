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
