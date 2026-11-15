// SPDX-License-Identifier: GPL-2.0-only
//! Shared coordinate equations for point and interval evaluation.
use oblikovati_solve::{ad::DualNumber, residual::ResidualError};

pub(super) trait DifferenceScalar: Sized {
    type Error;
    fn difference(&self, other: &Self) -> Result<Self, Self::Error>;
}
impl DifferenceScalar for DualNumber {
    type Error = ResidualError;
    fn difference(&self, other: &Self) -> Result<Self, Self::Error> {
        Ok(self.minus(other)?)
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
