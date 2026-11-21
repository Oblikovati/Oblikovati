// SPDX-License-Identifier: GPL-2.0-only
//! Derive conditional affine rows from shared sketch interval equations.
mod rows;
use super::{Sketch2D, SketchIntervalError, SketchIntervalTarget, SketchRelationId};
use crate::parameter::ParameterCollection;
use oblikovati_solve::affine_preference::{
    AffineLimits, AffinePreferenceCertificate, AffinePreferenceError, AffinePreferenceInput,
    CertifiedCoordinate, certify_affine_preference,
};
use std::{error::Error, fmt};

/// A local affine system conditional on interval arithmetic and supplied targets.
/// Projected document anchors are not included. Example: `system.bounds()`.
#[derive(Clone, Debug)]
pub struct ConditionalSketchAffineSystem {
    coefficients: Vec<Vec<f64>>,
    right_hand_side: Vec<f64>,
    relations: Vec<(SketchRelationId, usize)>,
    bounds: Vec<[f64; 2]>,
    anchor: Vec<f64>,
    targets: Vec<SketchIntervalTarget>,
}
impl ConditionalSketchAffineSystem {
    /// Read equation coefficients; example: `system.coefficients()`.
    pub fn coefficients(&self) -> &[Vec<f64>] {
        &self.coefficients
    }
    /// Read equation right sides; example: `system.right_hand_side()`.
    pub fn right_hand_side(&self) -> &[f64] {
        &self.right_hand_side
    }
    /// Read row identities; example: `system.relations()`.
    pub fn relations(&self) -> &[(SketchRelationId, usize)] {
        &self.relations
    }
    /// Read the finite domain for which the rows apply; example: `system.bounds()`.
    pub fn bounds(&self) -> &[[f64; 2]] {
        &self.bounds
    }
    /// Read the evaluation anchor; example: `system.anchor()`.
    pub fn anchor(&self) -> &[f64] {
        &self.anchor
    }
    /// Read caller-supplied target bounds; example: `system.targets()`.
    pub fn targets(&self) -> &[SketchIntervalTarget] {
        &self.targets
    }

    /// Certify the supplied point preferences when their optimum is inside the box.
    /// Example: `system.certify_optimum(&[(0, 1.0)], limits, || false)?`.
    /// The result remains conditional and does not include projected document anchors.
    pub fn certify_optimum(
        &self,
        preferences: &[(usize, f64)],
        limits: AffineLimits,
        cancelled: impl FnMut() -> bool,
    ) -> Result<ConditionalSketchAffineOptimum, SketchAffineError> {
        let input = AffinePreferenceInput {
            equations: &self.coefficients,
            right_hand_side: &self.right_hand_side,
            initial: &self.anchor,
            targets: preferences,
        };
        let certificate = certify_affine_preference(input, limits, cancelled)
            .map_err(SketchAffineError::Solve)?;
        check_domain(&self.bounds, certificate.coordinates())?;
        Ok(ConditionalSketchAffineOptimum {
            system: self.clone(),
            certificate,
        })
    }
}

/// An affine optimum inside the stated domain, under the system's assumptions.
/// This is a snapshot: callers must check source changes before using it.
#[derive(Clone, Debug)]
pub struct ConditionalSketchAffineOptimum {
    system: ConditionalSketchAffineSystem,
    certificate: AffinePreferenceCertificate,
}
impl ConditionalSketchAffineOptimum {
    /// Read the retained domain and source assumptions; example: `optimum.system()`.
    pub fn system(&self) -> &ConditionalSketchAffineSystem {
        &self.system
    }
    /// Read exact-optimum enclosures, not verified rounded geometry; example: `optimum.coordinates()`.
    pub fn coordinates(&self) -> &[CertifiedCoordinate] {
        self.certificate.coordinates()
    }
}

fn check_domain(
    bounds: &[[f64; 2]],
    coordinates: &[CertifiedCoordinate],
) -> Result<(), SketchAffineError> {
    if bounds.len() != coordinates.len() {
        return Err(SketchAffineError::Input(format!(
            "affine optimum has {} coordinates and {} domain intervals: expected equal lengths",
            coordinates.len(),
            bounds.len()
        )));
    }
    for (index, (domain, coordinate)) in bounds.iter().zip(coordinates).enumerate() {
        let output = coordinate.bounds();
        if output[0] < domain[0] || output[1] > domain[1] {
            return Err(SketchAffineError::OutsideDomain {
                index,
                output,
                domain: *domain,
            });
        }
    }
    Ok(())
}

/// Failure to derive conditional rows; example: `SketchAffineError::Input(message)`.
#[derive(Debug)]
pub enum SketchAffineError {
    Input(String),
    Interval(SketchIntervalError),
    Solve(AffinePreferenceError),
    OutsideDomain {
        index: usize,
        output: [f64; 2],
        domain: [f64; 2],
    },
    Indeterminate {
        relation: SketchRelationId,
        equation: usize,
        field: &'static str,
        bounds: [f64; 2],
    },
    Arithmetic(String),
}
impl fmt::Display for SketchAffineError {
    fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Input(message) | Self::Arithmetic(message) => out.write_str(message),
            Self::Interval(error) => write!(out, "sketch affine interval: {error}"),
            Self::Solve(error) => write!(out, "sketch affine optimum: {error}"),
            Self::OutsideDomain {
                index,
                output,
                domain,
            } => write!(
                out,
                "affine optimum coordinate {index} {output:?}: expected enclosure inside domain {domain:?}"
            ),
            Self::Indeterminate {
                relation,
                equation,
                field,
                bounds,
            } => write!(
                out,
                "affine row {relation:?}/{equation} {field} {bounds:?}: expected one finite binary64 value"
            ),
        }
    }
}
impl Error for SketchAffineError {}

impl Sketch2D {
    /// Derive local rows from interval equations over a finite box.
    /// Example: `sketch.conditional_affine_system(&parameters, &bounds, &anchor, &targets)?`.
    /// This is conditional evidence, not a sketch, target-source or drag certificate.
    pub fn conditional_affine_system(
        &self,
        parameters: &ParameterCollection<u64>,
        bounds: &[[f64; 2]],
        anchor: &[f64],
        targets: &[SketchIntervalTarget],
    ) -> Result<ConditionalSketchAffineSystem, SketchAffineError> {
        validate_anchor(bounds, anchor)?;
        let domain = self
            .interval_equations_with_targets(parameters, bounds, targets)
            .map_err(SketchAffineError::Interval)?;
        let point: Vec<_> = anchor.iter().map(|&value| [value, value]).collect();
        let at_anchor = self
            .interval_equations_with_targets(parameters, &point, targets)
            .map_err(SketchAffineError::Interval)?;
        let (coefficients, right_hand_side, relations) = rows::derive(&domain, &at_anchor, anchor)?;
        Ok(ConditionalSketchAffineSystem {
            coefficients,
            right_hand_side,
            relations,
            bounds: bounds.to_vec(),
            anchor: anchor.to_vec(),
            targets: targets.to_vec(),
        })
    }
}

fn validate_anchor(bounds: &[[f64; 2]], anchor: &[f64]) -> Result<(), SketchAffineError> {
    if bounds.len() != anchor.len() {
        return Err(SketchAffineError::Input(format!(
            "affine anchor has {} coordinates and box has {}: expected equal lengths",
            anchor.len(),
            bounds.len()
        )));
    }
    for (index, (&value, &[lo, hi])) in anchor.iter().zip(bounds).enumerate() {
        if !value.is_finite() || !lo.is_finite() || !hi.is_finite() || lo > value || value > hi {
            return Err(SketchAffineError::Input(format!(
                "affine anchor {index} value {value:?} and box [{lo:?}, {hi:?}]: expected finite ordered bounds containing the value"
            )));
        }
    }
    Ok(())
}
