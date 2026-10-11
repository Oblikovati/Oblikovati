// SPDX-License-Identifier: GPL-2.0-only
use super::{LineContactSide, circle_residual::positive_radius};
use oblikovati_solve::{ad::DualNumber, residual::ResidualError};
#[derive(Clone, Copy)]
pub(super) enum LengthVariableDomain {
    Positive,
    Signed(LineContactSide),
}
pub(super) fn scalar_length_target(
    value: &DualNumber,
    target: f64,
    domain: LengthVariableDomain,
) -> Result<Vec<DualNumber>, ResidualError> {
    let magnitude = match domain {
        LengthVariableDomain::Positive => {
            positive_radius(value)?;
            value.clone()
        }
        LengthVariableDomain::Signed(side) => value.scaled(if side == LineContactSide::Left {
            1.0
        } else {
            -1.0
        })?,
    };
    Ok(vec![magnitude.shifted(-target)?])
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn positive_length_domain_keeps_original_radius_residual_bits() {
        for value in [f64::MIN_POSITIVE, 1e-6, 1.0, 1e6] {
            let input = DualNumber::variable(value, 0, 2).unwrap();
            let expected = input.shifted(-0.5).unwrap();
            let actual = scalar_length_target(&input, 0.5, LengthVariableDomain::Positive).unwrap();
            assert_scalar_bits(&actual[0], &expected);
        }
    }
    #[test]
    fn signed_length_domains_match_abs_bits_and_keep_unit_rank_at_zero() {
        for (side, input) in [(LineContactSide::Left, 2.0), (LineContactSide::Right, -2.0)] {
            let value = DualNumber::variable(input, 0, 2).unwrap();
            let actual =
                scalar_length_target(&value, 0.5, LengthVariableDomain::Signed(side)).unwrap();
            let expected = value.abs().unwrap().shifted(-0.5).unwrap();
            assert_scalar_bits(&actual[0], &expected);
            let zero = DualNumber::variable(0.0, 0, 1).unwrap();
            let row = scalar_length_target(&zero, 0.0, LengthVariableDomain::Signed(side)).unwrap();
            assert_eq!(row[0].gradient()[0].abs(), 1.0);
        }
    }

    fn assert_scalar_bits(actual: &DualNumber, expected: &DualNumber) {
        assert_eq!(actual.value().to_bits(), expected.value().to_bits());
        let actual_bits: Vec<u64> = actual
            .gradient()
            .iter()
            .map(|value| value.to_bits())
            .collect();
        let expected_bits: Vec<u64> = expected
            .gradient()
            .iter()
            .map(|value| value.to_bits())
            .collect();
        assert_eq!(actual_bits, expected_bits);
    }
}
