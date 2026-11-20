// SPDX-License-Identifier: GPL-2.0-only
//! Resource and cancellation policy for exact arithmetic.
use super::{AffineLimits, AffinePreferenceError};
use num_rational::BigRational;

// These caps bound worst-case dense allocation and individual integer growth.
const MAX_ORDER: usize = 128;
const MAX_BITS: u64 = 4096;

pub(super) struct Policy<'a, F> {
    pub limits: AffineLimits,
    cancelled: &'a mut F,
}
impl<'a, F: FnMut() -> bool> Policy<'a, F> {
    pub fn new(limits: AffineLimits, cancelled: &'a mut F) -> Result<Self, AffinePreferenceError> {
        if limits.max_order == 0
            || limits.max_order > MAX_ORDER
            || limits.max_bits == 0
            || limits.max_bits > MAX_BITS
        {
            return Err(AffinePreferenceError::Input(format!(
                "affine limits {limits:?}: expected order 1..={MAX_ORDER} and bits 1..={MAX_BITS}"
            )));
        }
        let mut policy = Self { limits, cancelled };
        policy.poll()?;
        Ok(policy)
    }
    pub fn poll(&mut self) -> Result<(), AffinePreferenceError> {
        if (self.cancelled)() {
            return Err(AffinePreferenceError::Cancelled);
        }
        Ok(())
    }
    pub fn check(&self, value: &BigRational) -> Result<(), AffinePreferenceError> {
        let bits = value.numer().bits().max(value.denom().bits());
        if bits > self.limits.max_bits {
            return Err(AffinePreferenceError::Limit(format!(
                "rational size {bits} bits: expected at most {}",
                self.limits.max_bits
            )));
        }
        Ok(())
    }
}
