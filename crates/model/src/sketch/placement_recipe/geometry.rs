// SPDX-License-Identifier: GPL-2.0-only
use super::super::{
    SketchEntityId, SketchError, SketchPathMember, SketchPointInput, SketchPrimitiveCreation,
    path_member::indexed_entity,
};
/// Indexed placement geometry; a closed circle has no artificial path endpoints.
#[derive(Clone, Copy, Debug, PartialEq)]
pub enum SketchPlacementGeometry {
    Path(SketchPathMember),
    Ellipse {
        center: usize,
        major_axis: [f64; 2],
        radii: [f64; 2],
    },
    Circle {
        center: usize,
        radius: f64,
    },
}
impl SketchPlacementGeometry {
    pub(crate) fn primitive(
        self,
        points: &[SketchEntityId],
        construction: bool,
    ) -> Result<SketchPrimitiveCreation, SketchError> {
        match self {
            Self::Ellipse {
                center,
                major_axis,
                radii,
            } => Ok(SketchPrimitiveCreation::Ellipse {
                center: SketchPointInput::Owned(indexed_entity(points, center)?),
                major_axis,
                radii,
                construction,
            }),
            Self::Path(path) => path.primitive(points, construction),
            Self::Circle { center, radius } => Ok(SketchPrimitiveCreation::Circle {
                center: SketchPointInput::Owned(indexed_entity(points, center)?),
                radius,
                construction,
            }),
        }
    }
}
