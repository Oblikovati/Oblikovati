// SPDX-License-Identifier: GPL-2.0-only
//! Shared coordinate equations for point and interval evaluation.
use oblikovati_solve::{ad::DualNumber, residual::ResidualError};

pub(super) trait DifferenceScalar: Sized {
    type Error;
    fn difference(&self, other: &Self) -> Result<Self, Self::Error>;
    fn sum(&self, other: &Self) -> Result<Self, Self::Error>;
    fn half(&self) -> Result<Self, Self::Error>;
    fn absolute(&self) -> Result<Self, Self::Error>;
    fn subtract_target(&self, target: &Self) -> Result<Self, Self::Error>;
}
impl DifferenceScalar for DualNumber {
    type Error = ResidualError;
    fn difference(&self, other: &Self) -> Result<Self, Self::Error> {
        Ok(self.minus(other)?)
    }
    fn sum(&self, other: &Self) -> Result<Self, Self::Error> {
        Ok(self.plus(other)?)
    }
    fn half(&self) -> Result<Self, Self::Error> {
        Ok(self.scaled(0.5)?)
    }
    fn absolute(&self) -> Result<Self, Self::Error> {
        Ok(self.abs()?)
    }
    fn subtract_target(&self, target: &Self) -> Result<Self, Self::Error> {
        Ok(self.shifted(-target.value())?)
    }
}
pub(super) fn coincident<T: DifferenceScalar>(values: &[T]) -> Result<Vec<T>, T::Error> {
    Ok(vec![
        values[0].difference(&values[2])?,
        values[1].difference(&values[3])?,
    ])
}
pub(super) fn coordinate<T: DifferenceScalar>(
    values: &[T],
    axis: usize,
) -> Result<Vec<T>, T::Error> {
    Ok(vec![values[axis].difference(&values[axis + 2])?])
}

pub(super) fn fixed<T: DifferenceScalar>(values: &[T], targets: &[T]) -> Result<Vec<T>, T::Error> {
    values
        .iter()
        .zip(targets)
        .map(|(value, target)| value.subtract_target(target))
        .collect()
}
pub(super) fn midpoint<T: DifferenceScalar>(values: &[T]) -> Result<Vec<T>, T::Error> {
    (0..2)
        .map(|axis| {
            let middle = values[axis + 2].half()?.sum(&values[axis + 4].half()?)?;
            values[axis].difference(&middle)
        })
        .collect()
}
pub(super) fn axis_distance<T: DifferenceScalar>(values: &[T], axis: usize) -> Result<T, T::Error> {
    values[axis].difference(&values[axis + 2])?.absolute()
}
