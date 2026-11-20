// SPDX-License-Identifier: GPL-2.0-only
//! Exact affine preference certificates use explicit binary input assumptions.
use num_rational::BigRational;
use oblikovati_solve::affine_preference::{
    AffineLimits, AffinePreferenceError, AffinePreferenceInput, certify_affine_preference,
};

const LIMITS: AffineLimits = AffineLimits {
    max_order: 64,
    max_bits: 4096,
};

fn solve(
    rows: &[Vec<f64>],
    rhs: &[f64],
    initial: &[f64],
    targets: &[(usize, f64)],
) -> Result<oblikovati_solve::affine_preference::AffinePreferenceCertificate, AffinePreferenceError>
{
    certify_affine_preference(
        AffinePreferenceInput {
            equations: rows,
            right_hand_side: rhs,
            initial,
            targets,
        },
        LIMITS,
        || false,
    )
}

#[test]
fn bounds_enclose_non_dyadic_best_fit_and_preserve_free_coordinates() {
    let rows = vec![vec![3., 1., 0.], vec![6., 2., 0.]];
    let result = solve(&rows, &[0., 0.], &[0., 0., 42.], &[(0, 1.), (1, 0.)]).unwrap();
    let coordinates = result.coordinates();
    assert_eq!(coordinates.len(), 3);
    assert_eq!(coordinates[0].rounded(), 0.1);
    assert_eq!(coordinates[1].rounded(), -0.3);
    assert_eq!(coordinates[2].bounds(), [42., 42.]);
    assert!(coordinates[0].bounds()[0] < coordinates[0].bounds()[1]);
    assert!(coordinates[1].bounds()[0] < coordinates[1].bounds()[1]);
}

#[test]
fn inconsistent_equations_and_bad_inputs_are_explicit() {
    let rows = vec![vec![1.], vec![1.]];
    assert!(matches!(
        solve(&rows, &[0., 1.], &[0.], &[]),
        Err(AffinePreferenceError::Inconsistent)
    ));
    assert!(matches!(
        solve(&rows, &[0.], &[0.], &[]),
        Err(AffinePreferenceError::Input(_))
    ));
    assert!(matches!(
        solve(&[], &[], &[0.], &[(0, f64::NAN)]),
        Err(AffinePreferenceError::Input(_))
    ));
    assert!(matches!(
        solve(&[], &[], &[0.], &[(0, 1.), (0, 2.)]),
        Err(AffinePreferenceError::Input(_))
    ));
}

#[test]
fn cancellation_and_resource_limits_do_not_return_a_certificate() {
    let input = AffinePreferenceInput {
        equations: &[],
        right_hand_side: &[],
        initial: &[0., 0.],
        targets: &[(0, 1.)],
    };
    assert!(matches!(
        certify_affine_preference(input, LIMITS, || true),
        Err(AffinePreferenceError::Cancelled)
    ));
    let limit = AffineLimits {
        max_order: 1,
        max_bits: 4096,
    };
    assert!(matches!(
        certify_affine_preference(input, limit, || false),
        Err(AffinePreferenceError::Limit(_))
    ));
}

fn exact(value: f64) -> BigRational {
    BigRational::from_float(value).unwrap()
}

#[test]
fn enclosures_match_independent_two_coordinate_projection() {
    for a in -3..=3 {
        for b in -3..=3 {
            if a == 0 && b == 0 {
                continue;
            }
            for c in -2..=2 {
                let (a, b, c) = (a as f64, b as f64, c as f64);
                let rows = vec![vec![a, b]];
                let result = solve(&rows, &[c], &[0., 0.], &[(0, 2.), (1, -1.)]).unwrap();
                let scale = (exact(a) * exact(2.) - exact(b) - exact(c))
                    / (exact(a) * exact(a) + exact(b) * exact(b));
                let expected = [
                    exact(2.) - exact(a) * scale.clone(),
                    -exact(1.) - exact(b) * scale,
                ];
                for (coordinate, expected) in result.coordinates().iter().zip(expected) {
                    let [lower, upper] = coordinate.bounds();
                    assert!(
                        exact(lower) <= expected && expected <= exact(upper),
                        "{a}x + {b}y = {c}: {expected}"
                    );
                }
            }
        }
    }
}

#[test]
fn subnormal_optimum_is_enclosed_and_unrepresentable_output_is_declined() {
    let smallest = f64::from_bits(1);
    let result = solve(&[vec![2.]], &[smallest], &[0.], &[(0, 0.)]).unwrap();
    assert_eq!(result.coordinates()[0].bounds(), [0., smallest]);
    assert!(matches!(
        solve(&[vec![f64::MIN_POSITIVE]], &[f64::MAX], &[0.], &[(0, 0.)]),
        Err(AffinePreferenceError::Enclosure(_))
    ));
}

#[test]
fn bit_limits_and_late_cancellation_are_explicit() {
    let input = AffinePreferenceInput {
        equations: &[],
        right_hand_side: &[],
        initial: &[0.1],
        targets: &[],
    };
    let small = AffineLimits {
        max_order: 2,
        max_bits: 10,
    };
    assert!(matches!(
        certify_affine_preference(input, small, || false),
        Err(AffinePreferenceError::Limit(_))
    ));
    let rows = vec![vec![1., 0.], vec![0., 1.]];
    let input = AffinePreferenceInput {
        equations: &rows,
        right_hand_side: &[1., 2.],
        initial: &[0., 0.],
        targets: &[(0, 3.)],
    };
    let mut polls = 0;
    let result = certify_affine_preference(input, LIMITS, || {
        polls += 1;
        polls > 15
    });
    assert!(matches!(result, Err(AffinePreferenceError::Cancelled)));
}

#[test]
fn unconstrained_preferences_and_feasible_free_variables_are_deterministic() {
    let rows = vec![vec![1., 1.]];
    let first = solve(&rows, &[3.], &[0., 5.], &[]).unwrap();
    let second = solve(&rows, &[3.], &[0., 5.], &[]).unwrap();
    assert_eq!(first.coordinates(), second.coordinates());
    assert_eq!(first.coordinates()[0].bounds(), [-2., -2.]);
    assert_eq!(first.coordinates()[1].bounds(), [5., 5.]);
}

#[test]
fn malformed_rows_and_hard_limits_are_refused() {
    assert!(matches!(
        solve(&[vec![1., 2.]], &[0.], &[0.], &[]),
        Err(AffinePreferenceError::Input(_))
    ));
    assert!(matches!(
        solve(&[vec![f64::INFINITY]], &[0.], &[0.], &[]),
        Err(AffinePreferenceError::Input(_))
    ));
    assert!(matches!(
        solve(&[], &[], &[0.], &[(1, 1.)]),
        Err(AffinePreferenceError::Input(_))
    ));
    let input = AffinePreferenceInput {
        equations: &[],
        right_hand_side: &[],
        initial: &[0.],
        targets: &[],
    };
    let invalid = AffineLimits {
        max_order: 129,
        max_bits: 4096,
    };
    assert!(matches!(
        certify_affine_preference(input, invalid, || false),
        Err(AffinePreferenceError::Input(_))
    ));
}
