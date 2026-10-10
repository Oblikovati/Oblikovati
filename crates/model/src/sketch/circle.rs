// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchEntityId, SketchError};

/// Analytic circle with a shared center identity and one positive radius variable.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchCircle2D {
    pub id: SketchEntityId,
    pub center: SketchEntityId,
    pub radius: f64,
}
impl Sketch2D {
    /// Inspect circles in insertion order; example: `sketch.circles()`.
    pub fn circles(&self) -> &[SketchCircle2D] {
        &self.circles
    }
    /// Resolve a circle identity; example: `sketch.circle(SketchEntityId(9))?`.
    pub fn circle(&self, id: SketchEntityId) -> Result<&SketchCircle2D, SketchError> {
        self.circles
            .iter()
            .find(|circle| circle.id == id)
            .ok_or_else(|| {
                SketchError::Definition(format!(
                    "sketch circle {id:?}: expected an existing circle"
                ))
            })
    }
    /// Add an analytic circle in document working units; example: `sketch.add_circle(id, center, 2.0)?`.
    pub fn add_circle(
        &mut self,
        id: SketchEntityId,
        center: SketchEntityId,
        radius: f64,
    ) -> Result<(), SketchError> {
        self.require_new_entity(id)?;
        self.point(center)?;
        require_radius(id, radius)?;
        self.circles.push(SketchCircle2D { id, center, radius });
        Ok(())
    }
    /// Change a radius after validation; example: `sketch.set_radius(id, 3.0)?`.
    pub fn set_radius(&mut self, id: SketchEntityId, radius: f64) -> Result<(), SketchError> {
        require_radius(id, radius)?;
        self.circle(id)?;
        self.circles
            .iter_mut()
            .find(|circle| circle.id == id)
            .expect("validated circle")
            .radius = radius;
        Ok(())
    }
}
pub(super) fn require_radius(id: SketchEntityId, radius: f64) -> Result<(), SketchError> {
    if !radius.is_finite() || radius <= 0.0 {
        return Err(SketchError::Definition(format!(
            "sketch circle {id:?} radius {radius:?}: expected a positive finite working-unit length"
        )));
    }
    Ok(())
}
