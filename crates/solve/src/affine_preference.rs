// SPDX-License-Identifier: GPL-2.0-only
//! Exact best fit for explicitly supplied affine equations over binary64 inputs.
//!
//! The certificate is conditional on the supplied equations and inputs. It does
//! not establish that they represent a sketch or that rounded output satisfies
//! the sketch tolerance policy.
mod elimination;
mod enclosure;
mod equations;
mod policy;
use std::{error::Error, fmt};

/// Explicit affine equations and coordinate targets.
/// Initial values only select free variables; they do not minimize their displacement.
/// Example: `AffinePreferenceInput { equations: &rows, right_hand_side: &rhs, initial: &seed, targets: &targets }`.
#[derive(Clone, Copy)]
pub struct AffinePreferenceInput<'a> {
    pub equations: &'a [Vec<f64>],
    pub right_hand_side: &'a [f64],
    pub initial: &'a [f64],
    pub targets: &'a [(usize, f64)],
}

/// Per-call resource limits; example: `AffineLimits { max_order: 64, max_bits: 4096 }`.
/// Order is coordinates plus equations. Maximums are 128 and 4096 bits; neither
/// limit guarantees an elapsed time.
#[derive(Clone, Copy, Debug)]
pub struct AffineLimits {
    pub max_order: usize,
    pub max_bits: u64,
}

/// A bounded failure; example: `AffinePreferenceError::Cancelled`.
#[derive(Debug, PartialEq, Eq)]
pub enum AffinePreferenceError {
    Input(String),
    Limit(String),
    Inconsistent,
    Cancelled,
    Enclosure(String),
    Verification(String),
}
impl fmt::Display for AffinePreferenceError {
    fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Input(message)
            | Self::Limit(message)
            | Self::Enclosure(message)
            | Self::Verification(message) => out.write_str(message),
            Self::Inconsistent => out.write_str("affine equations: expected a consistent system"),
            Self::Cancelled => out.write_str("affine preference: cancelled"),
        }
    }
}
impl Error for AffinePreferenceError {}

/// Binary64 output and an enclosure of the exact optimizer coordinate.
/// The rounded value can have a nonzero constraint residual.
#[derive(Clone, Debug, PartialEq)]
pub struct CertifiedCoordinate {
    rounded: f64,
    bounds: [f64; 2],
}
impl CertifiedCoordinate {
    /// Read the rounded coordinate; example: `coordinate.rounded()`.
    pub fn rounded(&self) -> f64 {
        self.rounded
    }
    /// Read enclosing endpoints; example: `coordinate.bounds()`.
    pub fn bounds(&self) -> [f64; 2] {
        self.bounds
    }
}

/// Exact KKT evidence for the supplied equations, not for a model using them.
#[derive(Clone, Debug)]
pub struct AffinePreferenceCertificate {
    coordinates: Vec<CertifiedCoordinate>,
}
impl AffinePreferenceCertificate {
    /// Read the enclosed optimizer coordinates; example: `certificate.coordinates()`.
    pub fn coordinates(&self) -> &[CertifiedCoordinate] {
        &self.coordinates
    }
}

/// Certify a best fit for the exact binary values in `input`.
/// Example: `certify_affine_preference(input, limits, || false)?`.
pub fn certify_affine_preference(
    input: AffinePreferenceInput<'_>,
    limits: AffineLimits,
    mut cancelled: impl FnMut() -> bool,
) -> Result<AffinePreferenceCertificate, AffinePreferenceError> {
    let mut policy = policy::Policy::new(limits, &mut cancelled)?;
    let problem = equations::prepare(input, &mut policy)?;
    let (matrix, seed) = equations::kkt(&problem, &mut policy)?;
    let solution = elimination::solve(matrix, seed, &mut policy)?;
    equations::verify(&problem, &solution, &mut policy)?;
    let mut coordinates = Vec::with_capacity(problem.initial.len());
    for value in &solution[..problem.initial.len()] {
        policy.poll()?;
        coordinates.push(enclosure::coordinate(value)?);
    }
    Ok(AffinePreferenceCertificate { coordinates })
}
