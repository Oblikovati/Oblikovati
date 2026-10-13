// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchEntityId, SketchError};
use crate::sketch::SketchSelectionTransform;
/// A fresh selection translated by a vector or reflected in an owned line.
#[derive(Clone, Copy, Debug)]
pub enum SketchCopyOperation {
    Translate([f64; 2]),
    Mirror(SketchEntityId),
}
impl SketchCopyOperation {
    /// Name the authored history step; example: `SketchCopyOperation::Mirror(line).label()`.
    pub fn label(self) -> &'static str {
        match self {
            Self::Translate(_) => "Copy sketch entities",
            Self::Mirror(_) => "Mirror sketch entities",
        }
    }
    pub(super) fn resolve(
        self,
        source: &Sketch2D,
    ) -> Result<SketchSelectionTransform, SketchError> {
        match self {
            Self::Translate(vector) => Ok(SketchSelectionTransform::Move(vector)),
            Self::Mirror(id) => mirror_transform(source, id),
        }
    }
}
fn mirror_transform(
    source: &Sketch2D,
    id: SketchEntityId,
) -> Result<SketchSelectionTransform, SketchError> {
    let [first, last] = source.line(id)?.endpoints;
    let start = source.point(first)?.position;
    let end = source.point(last)?.position;
    if start == end {
        return Err(SketchError::Definition(format!(
            "mirror line {id:?} with endpoints {start:?}, {end:?}: expected distinct finite points"
        )));
    }
    Ok(SketchSelectionTransform::Reflect { start, end })
}
