// SPDX-License-Identifier: GPL-2.0-only
use super::super::{
    SketchEntityId, SketchError, SketchPathMember, SketchPointInput, SketchPrimitiveCreation,
    path_member::indexed_entity,
};
/// Indexed placement geometry; a closed circle has no artificial path endpoints.
#[derive(Clone, Copy, Debug, PartialEq)]
pub enum SketchPlacementGeometry {
    Point(usize),
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
pub(crate) enum PlacementPrimitive {
    Point(SketchEntityId),
    Curve(SketchPrimitiveCreation),
}
impl SketchPlacementGeometry {
    pub(crate) fn primitive(
        self,
        points: &[SketchEntityId],
        construction: bool,
    ) -> Result<PlacementPrimitive, SketchError> {
        let primitive = match self {
            Self::Point(index) => {
                if construction {
                    return Err(SketchError::Definition(format!(
                        "placement point {index}, construction {construction}: expected false because points have no curve role"
                    )));
                }
                return Ok(PlacementPrimitive::Point(indexed_entity(points, index)?));
            }
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
        }?;
        Ok(PlacementPrimitive::Curve(primitive))
    }
}

pub(super) fn closed_boundary(sides: usize) -> Vec<super::SketchPlacementMember> {
    (0..sides)
        .map(|i| super::SketchPlacementMember {
            geometry: SketchPlacementGeometry::Path(SketchPathMember::Line([i, (i + 1) % sides])),
            construction: false,
        })
        .collect()
}
