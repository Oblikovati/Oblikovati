// SPDX-License-Identifier: GPL-2.0-only
//! Enclose exact rational coordinates in adjacent finite binary64 values.
use super::{AffinePreferenceError, CertifiedCoordinate};
use num_rational::BigRational;
use num_traits::ToPrimitive;
use std::cmp::Ordering;

pub(super) fn coordinate(
    value: &BigRational,
) -> Result<CertifiedCoordinate, AffinePreferenceError> {
    let rounded = value
        .to_f64()
        .filter(|value| value.is_finite())
        .ok_or_else(|| {
            AffinePreferenceError::Enclosure(format!(
                "affine optimizer coordinate {value}: expected finite binary64 output"
            ))
        })?;
    let bounds = checked_bounds(value, rounded)?;
    Ok(CertifiedCoordinate { rounded, bounds })
}

fn checked_bounds(value: &BigRational, rounded: f64) -> Result<[f64; 2], AffinePreferenceError> {
    let exact = rational(rounded)?;
    let bounds = match exact.cmp(value) {
        Ordering::Equal => [rounded, rounded],
        Ordering::Less => [rounded, rounded.next_up()],
        Ordering::Greater => [rounded.next_down(), rounded],
    };
    if !bounds.iter().all(|value| value.is_finite())
        || rational(bounds[0])? > *value
        || rational(bounds[1])? < *value
    {
        return Err(AffinePreferenceError::Enclosure(format!(
            "affine optimizer rounded {rounded:?}: expected adjacent finite enclosing values"
        )));
    }
    Ok(bounds)
}
fn rational(value: f64) -> Result<BigRational, AffinePreferenceError> {
    BigRational::from_float(value).ok_or_else(|| {
        AffinePreferenceError::Enclosure(format!(
            "enclosure endpoint {value:?}: expected finite binary64"
        ))
    })
}
