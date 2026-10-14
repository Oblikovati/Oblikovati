// SPDX-License-Identifier: GPL-2.0-only
use super::super::{
    SketchEntityId, SketchError, SketchPathMember, SketchPointInput, SketchPrimitiveCreation,
    path_member::indexed_entity,
};
/// Indexed placement geometry; a closed circle has no artificial path endpoints.
#[derive(Clone, Copy, Debug, PartialEq)]
pub enum SketchPlacementGeometry {
    Path(SketchPathMember),
    Circle { center: usize, radius: f64 },
}
impl SketchPlacementGeometry {
    pub(crate) fn primitive(
        self,
        points: &[SketchEntityId],
        construction: bool,
    ) -> Result<SketchPrimitiveCreation, SketchError> {
        match self {
            Self::Path(path) => path.primitive(points, construction),
            Self::Circle { center, radius } => Ok(SketchPrimitiveCreation::Circle {
                center: SketchPointInput::Owned(indexed_entity(points, center)?),
                radius,
                construction,
            }),
        }
    }
}
