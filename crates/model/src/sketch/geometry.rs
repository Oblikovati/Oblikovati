// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchError};

/// Persistent local entity identity supplied by the owning document, never by solver order.
#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub struct SketchEntityId(pub u64);
/// Planar coordinates in the owning document's working length unit.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchPoint2D {
    pub id: SketchEntityId,
    pub position: [f64; 2],
}
/// A line references owned points; shared identity is structural coincidence.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct SketchLine2D {
    pub id: SketchEntityId,
    pub endpoints: [SketchEntityId; 2],
}
impl Sketch2D {
    /// Add a finite point without allocating a new identity; example: `sketch.add_point(SketchEntityId(1), [0.0, 0.0])?`.
    pub fn add_point(&mut self, id: SketchEntityId, position: [f64; 2]) -> Result<(), SketchError> {
        self.require_new_entity(id)?;
        require_position(position)?;
        self.points.push(SketchPoint2D { id, position });
        Ok(())
    }
    /// Add a line with shared endpoints; example: `sketch.add_line(SketchEntityId(3), [a, b])?`.
    pub fn add_line(
        &mut self,
        id: SketchEntityId,
        endpoints: [SketchEntityId; 2],
    ) -> Result<(), SketchError> {
        self.require_new_entity(id)?;
        for point in endpoints {
            self.point(point)?;
        }
        self.lines.push(SketchLine2D { id, endpoints });
        Ok(())
    }
    /// Move an authored point atomically; example: `sketch.set_position(a, [2.0, 3.0])?`.
    pub fn set_position(
        &mut self,
        id: SketchEntityId,
        position: [f64; 2],
    ) -> Result<(), SketchError> {
        require_position(position)?;
        self.point(id)?;
        self.points
            .iter_mut()
            .find(|point| point.id == id)
            .expect("validated point")
            .position = position;
        Ok(())
    }
    pub(super) fn require_new_entity(&self, id: SketchEntityId) -> Result<(), SketchError> {
        if id.0 == 0
            || self.points.iter().any(|point| point.id == id)
            || self.lines.iter().any(|line| line.id == id)
            || self.circles.iter().any(|circle| circle.id == id)
            || self.arcs.iter().any(|arc| arc.id == id)
        {
            return Err(SketchError::Definition(format!(
                "sketch entity {id:?}: expected a unique nonzero identity"
            )));
        }
        Ok(())
    }
}
pub(super) fn require_position(position: [f64; 2]) -> Result<(), SketchError> {
    if position.iter().any(|coordinate| !coordinate.is_finite()) {
        return Err(SketchError::Definition(format!(
            "sketch position {position:?}: expected two finite working-unit coordinates"
        )));
    }
    Ok(())
}
