// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchEntityId, SketchError, variables::SketchVariable};

#[derive(Clone, Copy)]
pub(super) enum CircularRadius {
    Independent(SketchEntityId),
    Derived(SketchEntityId),
}
#[derive(Clone, Copy)]
pub(super) struct CircularFrame {
    pub center: SketchEntityId,
    pub radius: CircularRadius,
    pub endpoints: Option<[SketchEntityId; 2]>,
}
impl Sketch2D {
    pub(super) fn circular_frame(&self, id: SketchEntityId) -> Result<CircularFrame, SketchError> {
        if let Some(circle) = self.circles.iter().find(|circle| circle.id == id) {
            return Ok(CircularFrame {
                center: circle.center,
                radius: CircularRadius::Independent(id),
                endpoints: None,
            });
        }
        if let Some(arc) = self.arcs.iter().find(|arc| arc.id == id) {
            return Ok(CircularFrame {
                center: arc.center,
                radius: CircularRadius::Derived(arc.start),
                endpoints: Some([arc.start, arc.end]),
            });
        }
        Err(SketchError::Definition(format!(
            "sketch circular curve {id:?}: expected an owned circle or arc"
        )))
    }
    pub(super) fn frame_variables(&self, frame: CircularFrame) -> Vec<SketchVariable> {
        let mut variables = self.point_variables(&[frame.center]);
        match frame.radius {
            CircularRadius::Independent(id) => variables.push(SketchVariable::Radius(id)),
            CircularRadius::Derived(start) => variables.extend(self.point_variables(&[start])),
        }
        variables
    }
    pub(super) fn frame_values(
        &self,
        frame: CircularFrame,
    ) -> Result<([f64; 2], f64), SketchError> {
        let center = self.point(frame.center)?.position;
        let radius = match frame.radius {
            CircularRadius::Independent(id) => self.circle(id)?.radius,
            CircularRadius::Derived(start) => {
                let start = self.point(start)?.position;
                (start[0] - center[0]).hypot(start[1] - center[1])
            }
        };
        Ok((center, radius))
    }
}
