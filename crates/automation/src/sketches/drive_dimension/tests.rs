// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn drive_limit_errors_name_the_correct_metric_unit_and_offending_bound() {
    for (category, units) in [
        (Dimension::Length, "centimetres"),
        (Dimension::Angle, "radians"),
    ] {
        let request = DriveDimensionArgs {
            min: f64::INFINITY,
            max: 1.0,
            ..Default::default()
        };
        let message = convert_limits(&request, category, 1.0).unwrap_err().0;
        assert!(message.contains("inf") && message.contains(units));
    }
}
#[test]
fn finite_wire_limits_that_overflow_working_units_are_refused() {
    let request = DriveDimensionArgs {
        min: 0.0,
        max: f64::MAX,
        ..Default::default()
    };
    let message = convert_limits(&request, Dimension::Length, 1e-6)
        .unwrap_err()
        .0;
    assert!(message.contains(&format!("{:?}", f64::MAX)));
    assert!(message.contains("representable in working units"));
}
