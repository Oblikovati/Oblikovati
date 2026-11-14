// SPDX-License-Identifier: GPL-2.0-only
use super::SketchEllipse2D;
use crate::sketch::{Sketch2D, SketchEntityId, SketchError};
impl Sketch2D {
    /// Inspect ellipses in insertion order; example: `sketch.ellipses()`.
    pub fn ellipses(&self) -> &[SketchEllipse2D] {
        &self.ellipses
    }
    /// Resolve an owned analytic ellipse; example: `sketch.ellipse(selected)?`.
    pub fn ellipse(&self, id: SketchEntityId) -> Result<&SketchEllipse2D, SketchError> {
        self.ellipses
            .iter()
            .find(|shape| shape.id() == id)
            .ok_or_else(|| {
                SketchError::Definition(format!(
                    "sketch ellipse {id:?}: expected an owned analytic ellipse"
                ))
            })
    }
    /// Adopt a validated ellipse with a shared owned center; example: `sketch.add_ellipse(value)?`.
    pub fn add_ellipse(&mut self, ellipse: SketchEllipse2D) -> Result<(), SketchError> {
        self.require_new_entity(ellipse.id())?;
        self.point(ellipse.center())?;
        self.ellipses.push(ellipse);
        self.entity_order.push(ellipse.id());
        Ok(())
    }
}
