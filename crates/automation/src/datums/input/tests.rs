// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn coordinate_validation_covers_missing_wrong_sized_and_nonfinite_components() {
    for values in [
        None,
        Some(&[1.0, 2.0][..]),
        Some(&[0.0, f64::NAN, 0.0][..]),
        Some(&[f64::INFINITY, 0.0, 0.0][..]),
    ] {
        let error = coordinates(values, "point position").unwrap_err();
        assert!(error.to_string().contains("exactly three finite"));
    }
    assert_eq!(
        coordinates(Some(&[1.0, 2.0, 3.0]), "point position").unwrap(),
        [1.0, 2.0, 3.0]
    );
}
