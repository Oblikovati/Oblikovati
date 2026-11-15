// SPDX-License-Identifier: GPL-2.0-only
use super::enclosure::Enclosure;
use crate::sketch::linear_equations::DifferenceScalar;

#[derive(Clone)]
pub(super) struct IntervalDual {
    pub value: Enclosure,
    pub gradient: Vec<Enclosure>,
}
impl IntervalDual {
    pub fn variable(value: Enclosure, index: usize, count: usize) -> Result<Self, String> {
        let zero = Enclosure::new([0.0, 0.0])?;
        let one = Enclosure::new([1.0, 1.0])?;
        let mut gradient = vec![zero; count];
        gradient[index] = one;
        Ok(Self { value, gradient })
    }
}
impl DifferenceScalar for IntervalDual {
    type Error = String;
    fn difference(&self, other: &Self) -> Result<Self, String> {
        let gradient = self
            .gradient
            .iter()
            .zip(&other.gradient)
            .map(|(a, b)| a.difference(*b))
            .collect::<Result<_, _>>()?;
        Ok(Self {
            value: self.value.difference(other.value)?,
            gradient,
        })
    }
}
