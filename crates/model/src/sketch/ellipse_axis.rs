// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_solve::{ad::DualNumber, residual::ResidualError};
pub(super) fn alignment_residual(
    orientation: &DualNumber,
    quarter_turn: bool,
) -> Result<Vec<DualNumber>, ResidualError> {
    let target = if quarter_turn {
        std::f64::consts::FRAC_PI_2
    } else {
        0.0
    };
    // #3788: an unoriented axis repeats at pi; angle error retains a gradient at perpendicular seeds.
    let doubled = orientation.shifted(-target)?.scaled(2.0)?;
    Ok(vec![doubled.sin()?.atan2(&doubled.cos()?)?.scaled(0.5)?])
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn periodic_axis_error_has_unit_derivative_at_aligned_and_perpendicular_seeds() {
        for quarter_turn in [false, true] {
            for angle in [
                0.0,
                0.3,
                -0.3,
                std::f64::consts::FRAC_PI_2,
                std::f64::consts::PI,
            ] {
                let residual =
                    alignment_residual(&DualNumber::seed(&[angle]).unwrap()[0], quarter_turn)
                        .unwrap();
                assert!((residual[0].gradient()[0] - 1.0).abs() < 1e-14);
                assert!(residual[0].value().abs() <= std::f64::consts::FRAC_PI_2);
            }
        }
    }
}
