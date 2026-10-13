// SPDX-License-Identifier: GPL-2.0-only
use super::{SketchEntityId, SketchError, SketchPointInput, SketchPrimitiveCreation};
/// An indexed line or arc; adjacent members can share defining points.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum SketchPathMember {
    Line([usize; 2]),
    Arc {
        center: usize,
        endpoints: [usize; 2],
        counter_clockwise: bool,
    },
}

impl SketchPathMember {
    pub(crate) fn primitive(
        self,
        points: &[SketchEntityId],
        construction: bool,
    ) -> Result<SketchPrimitiveCreation, SketchError> {
        let point = |index| indexed_entity(points, index).map(SketchPointInput::Owned);
        Ok(match self {
            Self::Line([a, b]) => SketchPrimitiveCreation::Line {
                endpoints: [point(a)?, point(b)?],
                construction,
                centerline: false,
            },
            Self::Arc {
                center,
                endpoints: [a, b],
                counter_clockwise,
            } => SketchPrimitiveCreation::Arc {
                center: point(center)?,
                endpoints: [point(a)?, point(b)?],
                counter_clockwise,
                construction,
            },
        })
    }
}
pub(crate) fn indexed_entity(
    ids: &[SketchEntityId],
    index: usize,
) -> Result<SketchEntityId, SketchError> {
    ids.get(index).copied().ok_or_else(|| {
        SketchError::Definition(format!(
            "placement index {index}: expected an index below {}",
            ids.len()
        ))
    })
}
