// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchEntityId, SketchError, variables::SketchVariable};

/// Circular arc with shared defining points and a retained authored sweep direction.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchArc2D {
    pub id: SketchEntityId,
    pub center: SketchEntityId,
    pub start: SketchEntityId,
    pub end: SketchEntityId,
    pub counter_clockwise: bool,
}
impl Sketch2D {
    /// Inspect arcs in insertion order; example: `sketch.arcs()`.
    pub fn arcs(&self) -> &[SketchArc2D] {
        &self.arcs
    }
    /// Resolve an owned arc; example: `sketch.arc(SketchEntityId(9))?`.
    pub fn arc(&self, id: SketchEntityId) -> Result<&SketchArc2D, SketchError> {
        self.arcs.iter().find(|arc| arc.id == id).ok_or_else(|| {
            SketchError::Definition(format!("sketch arc {id:?}: expected an existing arc"))
        })
    }
    /// Add shared arc references and structural circularity; example: `sketch.add_arc(id, center, [start, end], true)?`.
    pub fn add_arc(
        &mut self,
        id: SketchEntityId,
        center: SketchEntityId,
        endpoints: [SketchEntityId; 2],
        counter_clockwise: bool,
    ) -> Result<(), SketchError> {
        self.require_new_entity(id)?;
        for point in [center, endpoints[0], endpoints[1]] {
            self.point(point)?;
        }
        self.arcs.push(SketchArc2D {
            id,
            center,
            start: endpoints[0],
            end: endpoints[1],
            counter_clockwise,
        });
        self.entity_order.push(id);
        Ok(())
    }
    /// Derive radius from center and start without an independent scalar; example: `sketch.arc_radius(id)?`.
    pub fn arc_radius(&self, id: SketchEntityId) -> Result<f64, SketchError> {
        self.arc(id)?;
        let (_, radius) = self.frame_values(self.circular_frame(id)?)?;
        if !radius.is_finite() || radius <= 0.0 {
            return Err(SketchError::Definition(format!(
                "sketch arc {id:?} radius {radius:?}: expected a positive finite derived length"
            )));
        }
        Ok(radius)
    }
    pub(super) fn arc_columns(&self, arc: &SketchArc2D) -> Result<Vec<usize>, SketchError> {
        [arc.center, arc.start, arc.end]
            .into_iter()
            .flat_map(|point| {
                [
                    SketchVariable::Coordinate { point, axis: 0 },
                    SketchVariable::Coordinate { point, axis: 1 },
                ]
            })
            .map(|variable| self.variable_column(variable))
            .collect()
    }
}
