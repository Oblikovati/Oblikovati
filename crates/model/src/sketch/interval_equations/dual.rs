// SPDX-License-Identifier: GPL-2.0-only
use super::enclosure::Enclosure;
use crate::sketch::linear_equations::DifferenceScalar;

#[derive(Clone)]
pub(super) struct IntervalDual {
    pub value: Enclosure,
    pub gradient: Vec<Enclosure>,
}
impl IntervalDual {
    pub fn constant(value: Enclosure, count: usize) -> Result<Self, String> {
        let zero = Enclosure::new([0.0, 0.0])?;
        Ok(Self {
            value,
            gradient: vec![zero; count],
        })
    }
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
    fn sum(&self, other: &Self) -> Result<Self, String> {
        let gradient = self
            .gradient
            .iter()
            .zip(&other.gradient)
            .map(|(a, b)| a.sum(*b))
            .collect::<Result<_, _>>()?;
        Ok(Self {
            value: self.value.sum(other.value)?,
            gradient,
        })
    }
    fn half(&self) -> Result<Self, String> {
        let gradient = self
            .gradient
            .iter()
            .map(|item| item.half())
            .collect::<Result<_, _>>()?;
        Ok(Self {
            value: self.value.half()?,
            gradient,
        })
    }
    fn absolute(&self) -> Result<Self, String> {
        let [lower, upper] = self.value.bounds();
        if lower > 0.0 {
            return Ok(self.clone());
        }
        if upper < 0.0 {
            return Ok(Self {
                value: self.value.negate(),
                gradient: self.gradient.iter().map(|item| item.negate()).collect(),
            });
        }
        Err(format!(
            "absolute interval [{lower:?}, {upper:?}]: expected an interval strictly on one side of zero"
        ))
    }
    fn subtract_target(&self, target: &Self) -> Result<Self, String> {
        self.difference(target)
    }
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
