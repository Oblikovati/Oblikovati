// SPDX-License-Identifier: GPL-2.0-only
//! Project-owned boundary around directed binary64 subtraction.
use maryada::{Interval, IntervalOps};

#[derive(Clone, Copy)]
pub(in crate::sketch) struct Enclosure(Interval);
impl Enclosure {
    pub fn new(bounds: [f64; 2]) -> Result<Self, String> {
        if !bounds[0].is_finite() || !bounds[1].is_finite() || bounds[0] > bounds[1] {
            return Err(format!(
                "interval {bounds:?}: expected ordered finite endpoints"
            ));
        }
        Ok(Self(Interval::new(bounds[0], bounds[1])))
    }
    pub fn decimal(text: &str) -> Result<Self, String> {
        let input = format!("[{text}]");
        let value = input
            .parse::<Interval>()
            .map_err(|_| format!("decimal {text:?}: expected a finite decimal literal"))?;
        Self::finite(Self(value), "decimal")
    }
    pub fn include(self, value: f64) -> Result<Self, String> {
        let [lower, upper] = self.bounds();
        Self::new([lower.min(value), upper.max(value)])
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
    pub fn product(self, rhs: Self) -> Result<Self, String> {
        Self::finite(Self(self.0 * rhs.0), "product")
    }
    pub fn sum(self, rhs: Self) -> Result<Self, String> {
        Self::finite(Self(self.0 + rhs.0), "sum")
    }
    pub fn half(self) -> Result<Self, String> {
        Self::finite(Self(self.0 * Interval::new(0.5, 0.5)), "half")
    }
    pub fn negate(self) -> Self {
        Self(-self.0)
    }
    fn finite(value: Self, operation: &str) -> Result<Self, String> {
        let bounds = value.bounds();
        if !bounds.iter().all(|item| item.is_finite()) {
            return Err(format!(
                "{operation} {bounds:?}: expected finite enclosing endpoints"
            ));
        }
        Ok(value)
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
    #[test]
    fn directed_sum_and_half_contain_exact_binary_rationals() {
        let values = [
            -f64::MAX,
            -1.,
            -0.1,
            -f64::MIN_POSITIVE,
            -f64::from_bits(1),
            0.,
            f64::from_bits(1),
            f64::MIN_POSITIVE,
            0.1,
            1.,
            f64::MAX,
        ];
        for left in values {
            let half = Enclosure::new([left, left])
                .unwrap()
                .half()
                .unwrap()
                .bounds();
            let exact_half = rational(left) / rational(2.);
            assert!(rational(half[0]) <= exact_half && exact_half <= rational(half[1]));
            for right in values {
                let exact = rational(left) + rational(right);
                let result = Enclosure::new([left, left])
                    .unwrap()
                    .sum(Enclosure::new([right, right]).unwrap());
                let Ok(result) = result else {
                    assert!(exact > rational(f64::MAX) || exact < -rational(f64::MAX));
                    continue;
                };
                let [lower, upper] = result.bounds();
                assert!(rational(lower) <= exact && exact <= rational(upper));
            }
        }
    }
    #[test]
    fn directed_product_contains_exact_binary_rationals() {
        let values = [
            -f64::MAX,
            -1.,
            -0.1,
            -f64::MIN_POSITIVE,
            -f64::from_bits(1),
            0.,
            f64::from_bits(1),
            f64::MIN_POSITIVE,
            0.1,
            1.,
            f64::MAX,
        ];
        for left in values {
            for right in values {
                let exact = rational(left) * rational(right);
                let result = Enclosure::new([left, left])
                    .unwrap()
                    .product(Enclosure::new([right, right]).unwrap());
                let Ok(result) = result else {
                    assert!(exact > rational(f64::MAX) || exact < -rational(f64::MAX));
                    continue;
                };
                let [lower, upper] = result.bounds();
                assert!(rational(lower) <= exact && exact <= rational(upper));
            }
        }
    }
    #[test]
    fn directed_interval_products_contain_all_exact_corners() {
        let intervals = [
            [-2., -1.],
            [-0.1, 0.3],
            [0., f64::from_bits(1)],
            [0.1, 2.],
            [-f64::MIN_POSITIVE, f64::MIN_POSITIVE],
        ];
        for left in intervals {
            for right in intervals {
                let [lower, upper] = Enclosure::new(left)
                    .unwrap()
                    .product(Enclosure::new(right).unwrap())
                    .unwrap()
                    .bounds();
                for a in left {
                    for b in right {
                        let exact = rational(a) * rational(b);
                        assert!(rational(lower) <= exact && exact <= rational(upper));
                    }
                }
            }
        }
    }
    #[test]
    fn overflowing_decimal_source_is_refused() {
        assert!(Enclosure::decimal("1e9999").is_err());
        assert!(Enclosure::decimal("-1e9999").is_err());
    }
}
