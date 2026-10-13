// SPDX-License-Identifier: GPL-2.0-only
use super::SketchError;
use std::error::Error;
/// Caller-owned metric acceptance; no default or topology tolerance is supplied.
/// The bound uses the same working units as the supplied coordinates.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct CircumcircleOptions {
    pub maximum_point_residual: f64,
}
/// Owned native construction and measured input residuals, not a center-error certificate.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct PlanarCircumcircle {
    pub center: [f64; 2],
    pub radius: f64,
    pub counter_clockwise: bool,
    pub point_residuals: [f64; 3],
}
/// Current authored geometry supplied for verification after the owner's solve.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SolvedCircularSeed {
    pub center: [f64; 2],
    pub radius: f64,
    pub arc: Option<([[f64; 2]; 2], bool)>,
}
/// Kernel construction is injected; the model and automation do not link a geometry kernel.
pub trait SketchCircleConstruction {
    /// Create an isolated seed; example: `backend.circumcircle(points, options)?`.
    fn circumcircle(
        &self,
        points: [[f64; 2]; 3],
        options: CircumcircleOptions,
    ) -> Result<PlanarCircumcircle, Box<dyn Error>>;
    /// Check the final solved seed before adoption; example: `backend.verify(points, solved, options)?`.
    fn verify(
        &self,
        points: [[f64; 2]; 3],
        solved: SolvedCircularSeed,
        options: CircumcircleOptions,
    ) -> Result<(), Box<dyn Error>>;
}
/// Versioned authoring metric policy; the physical bound is independent of working units.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchConstructionPolicy {
    pub version: u32,
    pub maximum_point_residual_cm: f64,
}
impl Default for SketchConstructionPolicy {
    fn default() -> Self {
        // tol:metric — Go math.DefaultTolerance in database centimetres (aa8297d1, #3563).
        Self {
            version: 1,
            maximum_point_residual_cm: 1e-9,
        }
    }
}
impl SketchConstructionPolicy {
    /// Convert this explicit physical bound once; example: `policy.in_working_units(units.working_scale())?`.
    pub fn in_working_units(self, scale: f64) -> Result<CircumcircleOptions, SketchError> {
        let bound = self.maximum_point_residual_cm / scale;
        if !self.accepts_bound(scale, bound) {
            return Err(SketchError::Definition(format!(
                "construction policy {self:?}, scale {scale}: expected version 1, positive finite scale and a finite nonnegative representable bound"
            )));
        }
        Ok(CircumcircleOptions {
            maximum_point_residual: bound,
        })
    }
    fn accepts_bound(self, scale: f64, bound: f64) -> bool {
        self.version == 1
            && scale.is_finite()
            && scale > 0.0
            && bound.is_finite()
            && bound >= 0.0
            && (self.maximum_point_residual_cm == 0.0 || bound > 0.0)
            && self.maximum_point_residual_cm.is_finite()
            && self.maximum_point_residual_cm >= 0.0
    }
}
