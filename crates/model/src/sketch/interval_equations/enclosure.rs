// SPDX-License-Identifier: GPL-2.0-only
//! Project-owned boundary around directed binary64 subtraction.
use maryada::{Interval, IntervalOps};

#[derive(Clone, Copy)]
pub(super) struct Enclosure(Interval);
impl Enclosure {
    pub fn new(bounds: [f64; 2]) -> Result<Self, String> {
        if !bounds[0].is_finite() || !bounds[1].is_finite() || bounds[0] > bounds[1] {
            return Err(format!(
                "interval {bounds:?}: expected ordered finite endpoints"
            ));
        }
        Ok(Self(Interval::new(bounds[0], bounds[1])))
    }
    pub fn difference(self, rhs: Self) -> Result<Self, String> {
        let result = Self(self.0 - rhs.0);
        let bounds = result.bounds();
        if !bounds.iter().all(|value| value.is_finite()) {
            return Err(format!(
                "difference {bounds:?}: expected finite enclosing endpoints"
            ));
        }
        Ok(result)
    }
    pub fn bounds(self) -> [f64; 2] {
        let (lower, upper) = self.0.bounds();
        [lower, upper]
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use num_rational::BigRational;

    fn rational(value: f64) -> BigRational {
        BigRational::from_float(value).unwrap()
    }
    #[test]
    fn directed_subtraction_contains_exact_binary_rational_corners() {
        let values = [
            -f64::MAX,
            -1.0,
            -0.1,
            -f64::MIN_POSITIVE,
            -f64::from_bits(1),
            -0.0,
            0.0,
            f64::from_bits(1),
            f64::MIN_POSITIVE,
            0.1,
            1.0,
            f64::MAX,
        ];
        for left in values {
            for right in values {
                let actual = Enclosure::new([left, left])
                    .unwrap()
                    .difference(Enclosure::new([right, right]).unwrap());
                let exact = rational(left) - rational(right);
                let Some(result) = actual.ok() else {
                    assert!(exact > rational(f64::MAX) || exact < -rational(f64::MAX));
                    continue;
                };
                let [lo, hi] = result.bounds();
                assert!(
                    rational(lo) <= exact && exact <= rational(hi),
                    "{left:?} - {right:?}: [{lo:?}, {hi:?}]"
                );
            }
        }
    }
}
