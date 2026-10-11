// SPDX-License-Identifier: GPL-2.0-only
use super::{
    Sketch2D, SketchEntityId, SketchError, circle::require_radius_value, geometry::require_position,
};

/// A defining point is new in working units or shares an existing point in this sketch.
#[derive(Clone, Copy, Debug, PartialEq)]
pub enum SketchPointInput {
    New([f64; 2]),
    Owned(SketchEntityId),
}
/// Typed primitive inputs; curve construction roles are explicit and plain points have none.
#[derive(Clone, Debug, PartialEq)]
pub enum SketchPrimitiveCreation {
    Point {
        position: [f64; 2],
    },
    Line {
        endpoints: [SketchPointInput; 2],
        construction: bool,
    },
    Circle {
        center: SketchPointInput,
        radius: f64,
        construction: bool,
    },
    Arc {
        center: SketchPointInput,
        endpoints: [SketchPointInput; 2],
        counter_clockwise: bool,
        construction: bool,
    },
}
impl SketchPrimitiveCreation {
    pub(crate) fn defining_points(&self) -> Vec<SketchPointInput> {
        match self {
            Self::Point { position } => vec![SketchPointInput::New(*position)],
            Self::Line { endpoints, .. } => endpoints.to_vec(),
            Self::Circle { center, .. } => vec![*center],
            Self::Arc {
                center, endpoints, ..
            } => vec![*center, endpoints[0], endpoints[1]],
        }
    }
    pub(crate) fn validate(&self, geometry: &Sketch2D) -> Result<(), SketchError> {
        for point in self.defining_points() {
            match point {
                SketchPointInput::New(position) => require_position(position)?,
                SketchPointInput::Owned(id) => {
                    geometry.point(id)?;
                }
            }
        }
        if let Self::Circle { radius, .. } = self {
            require_radius_value(*radius)?;
        }
        Ok(())
    }
    pub(crate) fn append_curve(
        &self,
        geometry: &mut Sketch2D,
        id: SketchEntityId,
        points: &[SketchEntityId],
    ) -> Result<(), SketchError> {
        match self {
            Self::Point { .. } => return Ok(()),
            Self::Line { .. } => geometry.add_line(id, [points[0], points[1]])?,
            Self::Circle { radius, .. } => geometry.add_circle(id, points[0], *radius)?,
            Self::Arc {
                counter_clockwise, ..
            } => geometry.add_arc(id, points[0], [points[1], points[2]], *counter_clockwise)?,
        }
        geometry.set_curve_construction(&[id], self.construction())?;
        Ok(())
    }
    fn construction(&self) -> bool {
        match self {
            Self::Point { .. } => false,
            Self::Line { construction, .. }
            | Self::Circle { construction, .. }
            | Self::Arc { construction, .. } => *construction,
        }
    }
}
