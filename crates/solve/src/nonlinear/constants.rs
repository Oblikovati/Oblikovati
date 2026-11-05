// SPDX-License-Identifier: GPL-2.0-only
use super::SolveError;
use std::collections::BTreeSet;

pub(super) fn seed(
    values: &[f64],
    constants: &[(usize, f64)],
) -> Result<(Vec<f64>, BTreeSet<usize>), SolveError> {
    let mut seeded = values.to_vec();
    let mut columns = BTreeSet::new();
    for &(column, value) in constants {
        if column >= values.len() || !value.is_finite() || !columns.insert(column) {
            return Err(SolveError::Options(format!(
                "fixed coordinate ({column}, {value:?}): expected a unique column below {} and a finite value",
                values.len()
            )));
        }
        seeded[column] = value;
    }
    Ok((seeded, columns))
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn constants_preserve_bits_and_refuse_invalid_or_duplicate_bindings() {
        let (values, columns) = seed(&[2., 3.], &[(0, -0.)]).unwrap();
        assert_eq!(values[0].to_bits(), (-0.0_f64).to_bits());
        assert_eq!(values[1], 3.);
        assert_eq!(columns, BTreeSet::from([0]));
        for constants in [vec![(2, 1.)], vec![(0, f64::NAN)], vec![(0, 1.), (0, 1.)]] {
            assert!(
                seed(&[2., 3.], &constants)
                    .unwrap_err()
                    .to_string()
                    .contains("expected a unique column below 2")
            );
        }
    }
}
