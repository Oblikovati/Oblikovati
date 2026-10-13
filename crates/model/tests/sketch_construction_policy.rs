// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::sketch::SketchConstructionPolicy;
#[test]
fn construction_policy_preserves_its_physical_metric_bound() {
    let policy = SketchConstructionPolicy::default();
    assert_eq!(policy.version, 1);
    assert_eq!(policy.maximum_point_residual_cm, 1e-9);
    for scale in [1.0, 0.001, 100.0] {
        let options = policy.in_working_units(scale).unwrap();
        assert_eq!(
            options.maximum_point_residual,
            policy.maximum_point_residual_cm / scale
        );
    }
    assert_eq!(
        SketchConstructionPolicy {
            maximum_point_residual_cm: 0.0,
            ..policy
        }
        .in_working_units(1.0)
        .unwrap()
        .maximum_point_residual,
        0.0
    );
}
#[test]
fn invalid_construction_policy_and_unrepresentable_conversion_fail_by_value() {
    let policy = SketchConstructionPolicy::default();
    for scale in [0.0, -1.0, f64::NAN, f64::INFINITY, f64::from_bits(1)] {
        let error = policy.in_working_units(scale).unwrap_err();
        assert!(error.to_string().contains("construction policy"));
    }
    for value in [-1.0, f64::NAN, f64::INFINITY] {
        assert!(
            SketchConstructionPolicy {
                maximum_point_residual_cm: value,
                ..policy
            }
            .in_working_units(1.0)
            .is_err()
        );
    }
    assert!(
        SketchConstructionPolicy {
            version: 2,
            ..policy
        }
        .in_working_units(1.0)
        .is_err()
    );
}

#[test]
fn positive_construction_bound_cannot_underflow_to_exact_zero() {
    let policy = SketchConstructionPolicy {
        maximum_point_residual_cm: f64::from_bits(1),
        ..Default::default()
    };
    assert!(policy.in_working_units(2.0).is_err());
    assert!(
        SketchConstructionPolicy {
            maximum_point_residual_cm: -f64::from_bits(1),
            ..policy
        }
        .in_working_units(2.0)
        .is_err()
    );
}
