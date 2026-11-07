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
