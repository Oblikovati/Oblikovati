// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{
    SKETCH_ENTITY_CONTROL_POINT_SPLINE, SKETCH_ENTITY_OFFSET_SPLINE, SPLINE_FIT_CHORD,
    SPLINE_FIT_SMOOTH, SPLINE_FIT_SWEET,
};
use oblikovati_model::sketch::{SketchEntityId, SketchSplineKind};

pub(super) fn prepare(
    snapshot: &DocumentSnapshot,
    request: &AddSketchEntityArgs,
) -> Result<PreparedEntity, Error> {
    if request.kind == SKETCH_ENTITY_OFFSET_SPLINE.0 {
        return offset(snapshot, request);
    }
    let positions = points::resolve(snapshot, request)?;
    let control =
        request.kind == SKETCH_ENTITY_CONTROL_POINT_SPLINE.0 || request.variant == "controlPoint";
    let kind = spline_kind(control, &request.fit_method)?;
    Ok(PreparedEntity::Primitive(
        SketchPrimitiveCreation::Spline {
            points: positions.into_iter().map(SketchPointInput::New).collect(),
            kind,
            closed: request.closed,
            construction: request.construction,
        },
        if control {
            SKETCH_ENTITY_CONTROL_POINT_SPLINE.0.as_ref()
        } else {
            SKETCH_ENTITY_SPLINE.0.as_ref()
        },
    ))
}

fn spline_kind(control: bool, method: &str) -> Result<SketchSplineKind, Error> {
    if control {
        if method.is_empty() {
            return Ok(SketchSplineKind::Control);
        }
        return Err(Error(format!(
            "control spline fit method {method:?}: expected empty for control intent"
        )));
    }
    let method = match method {
        "" | "smooth" => SPLINE_FIT_SMOOTH,
        "chord" => SPLINE_FIT_CHORD,
        "sweet" => SPLINE_FIT_SWEET,
        _ => {
            return Err(Error(format!(
                "spline fit method {method:?}: expected smooth, chord, sweet, or empty"
            )));
        }
    };
    Ok(SketchSplineKind::Fit { method })
}

fn offset(
    snapshot: &DocumentSnapshot,
    request: &AddSketchEntityArgs,
) -> Result<PreparedEntity, Error> {
    let refs = request.entity_refs.as_deref().unwrap_or_default();
    let [parent] = refs else {
        return Err(Error(format!(
            "offset spline references {refs:?}: expected exactly one parent spline ID"
        )));
    };
    let distance = length_expression(snapshot, &request.radius)?;
    Ok(PreparedEntity::Primitive(
        SketchPrimitiveCreation::OffsetSpline {
            parent: SketchEntityId(*parent),
            distance,
            construction: request.construction,
        },
        SKETCH_ENTITY_OFFSET_SPLINE.0.as_ref(),
    ))
}
