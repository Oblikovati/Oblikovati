// SPDX-License-Identifier: GPL-2.0-only
mod owned;
mod storage;
pub(super) mod variables;
use super::{SketchEntityId, SketchError, geometry::require_position};
use storage::EllipseDefinition;

/// Validated analytic ellipse value; sketch ownership is checked separately on adoption.
#[derive(Clone, Copy, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(try_from = "EllipseDefinition")]
pub struct SketchEllipse2D {
    id: SketchEntityId,
    center: SketchEntityId,
    major_axis: [f64; 2],
    radii: [f64; 2],
}
/// Analytic position and first derivative in document working units.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchEllipseEvaluation {
    pub position: [f64; 2],
    pub tangent: [f64; 2],
}
impl SketchEllipse2D {
    /// Prepare an ellipse without allocating identities; example: `SketchEllipse2D::new(id, center, [1.0, 0.0], [3.0, 1.0])?`.
    pub fn new(
        id: SketchEntityId,
        center: SketchEntityId,
        major_axis: [f64; 2],
        radii: [f64; 2],
    ) -> Result<Self, SketchError> {
        require_ellipse_identities(id, center)?;
        require_ellipse_radii(radii)?;
        Ok(Self {
            id,
            center,
            major_axis: normalized_ellipse_axis(major_axis)?,
            radii,
        })
    }
    /// Inspect the reserved curve identity; example: `ellipse.id()` when adding an owned entity.
    pub fn id(self) -> SketchEntityId {
        self.id
    }
    /// Inspect its shared center reference; example: `sketch.point(ellipse.center())?`.
    pub fn center(self) -> SketchEntityId {
        self.center
    }
    /// Inspect the normalized analytic frame; example: `ellipse.major_axis()` for serialization.
    pub fn major_axis(self) -> [f64; 2] {
        self.major_axis
    }
    /// Inspect major and minor lengths; example: `let [major, minor] = ellipse.radii();`.
    pub fn radii(self) -> [f64; 2] {
        self.radii
    }
    /// Seed one rotational degree of freedom; example: `ellipse.orientation()` for a solver variable.
    pub fn orientation(self) -> f64 {
        self.major_axis[1].atan2(self.major_axis[0])
    }
    /// Prepare radii without changing the analytic frame; example: `ellipse.with_radii([4.0, 1.0])?`.
    pub fn with_radii(self, radii: [f64; 2]) -> Result<Self, SketchError> {
        require_ellipse_radii(radii)?;
        Ok(Self { radii, ..self })
    }
    /// Prepare a rotation without introducing an axis-magnitude freedom; example: `ellipse.with_orientation(angle)?`.
    pub fn with_orientation(self, angle: f64) -> Result<Self, SketchError> {
        require_ellipse_angle(self.id, angle)?;
        // #1879: an unchanged solve must not round-trip its axis through trigonometry.
        if angle.to_bits() == self.orientation().to_bits() {
            return Ok(self);
        }
        let (sin, cos) = angle.sin_cos();
        Ok(Self {
            major_axis: [cos, sin],
            ..self
        })
    }
    pub(super) fn with_transformed_axis(self, major_axis: [f64; 2]) -> Result<Self, SketchError> {
        if super::same_components(self.major_axis, major_axis) {
            return Ok(self);
        }
        // Keep the existing unit-axis invariant under repeated floating-point rotations.
        Ok(Self {
            major_axis: normalized_ellipse_axis(major_axis)?,
            ..self
        })
    }
    /// Evaluate analytic position and tangent, not a tessellation; example: `ellipse.evaluate([0.0, 0.0], angle)?`.
    pub fn evaluate(
        self,
        center: [f64; 2],
        parameter: f64,
    ) -> Result<SketchEllipseEvaluation, SketchError> {
        require_position(center)?;
        require_ellipse_angle(self.id, parameter)?;
        let (sin, cos) = parameter.sin_cos();
        let offset =
            ellipse_frame_vector(self.major_axis, [self.radii[0] * cos, self.radii[1] * sin]);
        let tangent =
            ellipse_frame_vector(self.major_axis, [-self.radii[0] * sin, self.radii[1] * cos]);
        let position = [center[0] + offset[0], center[1] + offset[1]];
        require_ellipse_evaluation(self.id, parameter, position, tangent)?;
        Ok(SketchEllipseEvaluation { position, tangent })
    }
}
fn require_ellipse_identities(
    id: SketchEntityId,
    center: SketchEntityId,
) -> Result<(), SketchError> {
    if id.0 == 0 || center.0 == 0 || id == center {
        return Err(SketchError::Definition(format!(
            "ellipse identity {id:?} center {center:?}: expected distinct nonzero curve and point identities"
        )));
    }
    Ok(())
}
fn require_ellipse_radii(radii: [f64; 2]) -> Result<(), SketchError> {
    if radii
        .iter()
        .any(|radius| !radius.is_finite() || *radius <= 0.0)
        || radii[0] < radii[1]
    {
        return Err(SketchError::Definition(format!(
            "ellipse radii {radii:?}: expected finite major >= minor > 0 in working units"
        )));
    }
    Ok(())
}
fn normalized_ellipse_axis(axis: [f64; 2]) -> Result<[f64; 2], SketchError> {
    let scale = axis[0].abs().max(axis[1].abs());
    if axis.iter().any(|value| !value.is_finite()) || scale == 0.0 {
        return Err(SketchError::Definition(format!(
            "ellipse major axis {axis:?}: expected two finite components with nonzero length"
        )));
    }
    // Scale before normalization so both subnormal and maximum finite directions keep their angle.
    let scaled = [axis[0] / scale, axis[1] / scale];
    let length = scaled[0].hypot(scaled[1]);
    Ok([scaled[0] / length, scaled[1] / length])
}
fn require_ellipse_angle(id: SketchEntityId, angle: f64) -> Result<(), SketchError> {
    if !angle.is_finite() {
        return Err(SketchError::Definition(format!(
            "ellipse {id:?} angle {angle:?}: expected a finite radian value"
        )));
    }
    Ok(())
}
fn ellipse_frame_vector(axis: [f64; 2], components: [f64; 2]) -> [f64; 2] {
    [
        axis[0] * components[0] - axis[1] * components[1],
        axis[1] * components[0] + axis[0] * components[1],
    ]
}
fn require_ellipse_evaluation(
    id: SketchEntityId,
    parameter: f64,
    position: [f64; 2],
    tangent: [f64; 2],
) -> Result<(), SketchError> {
    if position
        .iter()
        .chain(&tangent)
        .any(|value| !value.is_finite())
    {
        return Err(SketchError::Definition(format!(
            "ellipse {id:?} parameter {parameter:?} position {position:?} tangent {tangent:?}: expected finite analytic coordinates and derivative"
        )));
    }
    Ok(())
}

pub(super) fn require_ellipse_frame(axis: [f64; 2], radii: [f64; 2]) -> Result<(), SketchError> {
    require_ellipse_radii(radii)?;
    normalized_ellipse_axis(axis)?;
    Ok(())
}
