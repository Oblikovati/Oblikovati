// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn valid() -> PrismResolution {
    PrismResolution {
        version: 1,
        weld: 1e-6,
        sew: 1e-7,
        angular: 1e-9,
        display_chordal: 1e-3,
        display_angular: 0.1,
        maximum_tolerance: 1e-5,
    }
}
#[test]
fn explicit_limits_preserve_each_value_and_refuse_an_invalid_cap() {
    let limits = valid().validate().unwrap();
    let kernel = limits.kernel();
    assert_eq!(
        (kernel.weld, kernel.sew, kernel.angular),
        (limits.weld, limits.sew, limits.angular)
    );
    assert_eq!(kernel.display.chordal, limits.display_chordal);
    assert_eq!(kernel.display.angular, limits.display_angular);
    assert_eq!(kernel.cap, ToleranceCap::Refuse(limits.maximum_tolerance));
    let cause = PrismResolution {
        maximum_tolerance: 0.,
        ..limits
    }
    .validate()
    .unwrap_err();
    assert!(cause.contains("cap=0"), "{cause}");
    assert!(
        PrismResolution {
            version: 2,
            ..limits
        }
        .validate()
        .unwrap_err()
        .contains("version 2")
    );
}
#[test]
fn model_relative_policy_scales_modelling_limits_but_not_angles_or_display() {
    let reference = valid();
    let policy = PrismResolutionPolicy::ModelRelative {
        reference,
        reference_size: 2.,
        minimum_size: 0.5,
    };
    let doubled = policy.for_size(4.).unwrap();
    assert_eq!(
        (doubled.weld, doubled.sew, doubled.maximum_tolerance),
        (
            reference.weld * 2.,
            reference.sew * 2.,
            reference.maximum_tolerance * 2.
        )
    );
    assert_eq!(
        (
            doubled.angular,
            doubled.display_chordal,
            doubled.display_angular
        ),
        (
            reference.angular,
            reference.display_chordal,
            reference.display_angular
        )
    );
    assert_eq!(policy.for_size(0.1).unwrap().weld, reference.weld * 0.25);
    assert!(policy.for_size(f64::NAN).is_err());
    let overflowing = PrismResolutionPolicy::ModelRelative {
        reference,
        reference_size: f64::MIN_POSITIVE,
        minimum_size: 1.,
    };
    assert!(overflowing.for_size(f64::MAX).is_err());
    assert!(
        PrismResolutionPolicy::ModelRelative {
            reference,
            reference_size: 0.,
            minimum_size: 1.
        }
        .validate()
        .is_err()
    );
    assert_eq!(
        PrismResolutionPolicy::Fixed(reference)
            .for_size(2.)
            .unwrap(),
        reference
    );
}
