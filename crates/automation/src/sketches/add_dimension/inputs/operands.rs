// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[derive(Clone, Copy, Debug)]
pub(super) enum Shape {
    Point,
    Line,
    Circular,
    Arc,
    Ellipse,
}
impl Shape {
    fn matches(self, kind: SketchEntityKind) -> bool {
        match self {
            Self::Point => kind == SKETCH_ENTITY_POINT,
            Self::Line => kind == SKETCH_ENTITY_LINE,
            Self::Circular => kind == SKETCH_ENTITY_CIRCLE || kind == SKETCH_ENTITY_ARC,
            Self::Arc => kind == SKETCH_ENTITY_ARC,
            Self::Ellipse => kind == SKETCH_ENTITY_ELLIPSE,
        }
    }
}
pub(super) fn require<const N: usize>(
    geometry: &Sketch2D,
    request: &AddDimensionArgs,
    shapes: [Shape; N],
) -> Result<[SketchEntityId; N], Error> {
    let refs = request.entities.as_deref().unwrap_or_default();
    let ids = <[u64; N]>::try_from(refs).map_err(|_| {
        Error(format!(
            "dimension {:?} entities {refs:?}: expected {N} owned operands {shapes:?}",
            request.kind
        ))
    })?;
    let mut checked = [SketchEntityId(0); N];
    for (index, (id, shape)) in ids.into_iter().zip(shapes).enumerate() {
        checked[index] = require_shape(geometry, &request.kind, id, shape)?;
    }
    Ok(checked)
}
fn require_shape(
    geometry: &Sketch2D,
    constructor: &str,
    id: u64,
    expected: Shape,
) -> Result<SketchEntityId, Error> {
    let id = SketchEntityId(id);
    let kind = geometry.entity_kind(id).map_err(|cause| {
        Error(format!(
            "dimension {constructor:?} entity {id:?}: expected owned {expected:?}: {cause}"
        ))
    })?;
    if !expected.matches(kind.clone()) {
        return Err(Error(format!(
            "dimension {constructor:?} entity {id:?} kind {kind:?}: expected owned {expected:?}"
        )));
    }
    Ok(id)
}
