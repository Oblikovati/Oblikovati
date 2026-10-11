// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{SKETCH_ENTITY_ARC, SKETCH_ENTITY_CIRCLE, SKETCH_ENTITY_POINT};
use oblikovati_model::{
    quantity::Dimension,
    sketch::{SketchPointInput, SketchPrimitiveCreation},
};
mod points;
#[derive(Clone, Copy)]
enum PrimitiveConstructor {
    StandalonePoint,
    CenterRadiusCircle,
    CenterStartEndArc,
}
pub(super) fn prepare(
    snapshot: &DocumentSnapshot,
    request: &AddSketchEntityArgs,
) -> Result<(SketchPrimitiveCreation, &'static str), Error> {
    let constructor = classify_kind(&request.kind)?;
    require_variant(constructor, &request.variant)?;
    let points = points::resolve(snapshot, request)?;
    require_point_count(constructor, &points)?;
    let input = construct_input(constructor, snapshot, &points, request)?;
    Ok((input, constructor.kind()))
}
fn classify_kind(kind: &str) -> Result<PrimitiveConstructor, Error> {
    match kind {
        kind if kind == SKETCH_ENTITY_POINT.0 => Ok(PrimitiveConstructor::StandalonePoint),
        kind if kind == SKETCH_ENTITY_CIRCLE.0 => Ok(PrimitiveConstructor::CenterRadiusCircle),
        kind if kind == SKETCH_ENTITY_ARC.0 => Ok(PrimitiveConstructor::CenterStartEndArc),
        _ => Err(Error(format!(
            "sketch.addEntity kind {kind:?}: expected a ported point, circle, or arc constructor; line inference and other kinds are not ported"
        ))),
    }
}
fn require_variant(constructor: PrimitiveConstructor, variant: &str) -> Result<(), Error> {
    let expected = constructor.variant();
    if !variant.is_empty() && variant != expected {
        return Err(Error(format!(
            "sketch.addEntity {} variant {variant:?}: expected empty or {expected:?}; other overloads are not ported",
            constructor.kind()
        )));
    }
    Ok(())
}
fn require_point_count(
    constructor: PrimitiveConstructor,
    points: &[[f64; 2]],
) -> Result<(), Error> {
    let expected = if matches!(constructor, PrimitiveConstructor::CenterStartEndArc) {
        3
    } else {
        1
    };
    if points.len() != expected {
        return Err(Error(format!(
            "sketch.addEntity {} points {points:?}: expected exactly {expected} defining points",
            constructor.kind()
        )));
    }
    Ok(())
}
fn length_expression(snapshot: &DocumentSnapshot, source: &str) -> Result<f64, Error> {
    snapshot.parameters.evaluate_scalar_expression(source, Dimension::Length).map(|quantity| quantity.value).map_err(|cause| Error(format!("sketch.addEntity length expression {source:?}: expected a healthy finite length: {cause}")))
}
impl PrimitiveConstructor {
    fn kind(self) -> &'static str {
        match self {
            Self::StandalonePoint => SKETCH_ENTITY_POINT.0.as_ref(),
            Self::CenterRadiusCircle => SKETCH_ENTITY_CIRCLE.0.as_ref(),
            Self::CenterStartEndArc => SKETCH_ENTITY_ARC.0.as_ref(),
        }
    }
    fn variant(self) -> &'static str {
        match self {
            Self::StandalonePoint => "",
            Self::CenterRadiusCircle => "centerRadius",
            Self::CenterStartEndArc => "centerStartEnd",
        }
    }
}

fn construct_input(
    constructor: PrimitiveConstructor,
    snapshot: &DocumentSnapshot,
    points: &[[f64; 2]],
    request: &AddSketchEntityArgs,
) -> Result<SketchPrimitiveCreation, Error> {
    Ok(match constructor {
        PrimitiveConstructor::StandalonePoint => SketchPrimitiveCreation::Point {
            position: points[0],
        },
        PrimitiveConstructor::CenterRadiusCircle => SketchPrimitiveCreation::Circle {
            center: SketchPointInput::New(points[0]),
            radius: length_expression(snapshot, &request.radius)?,
            construction: request.construction,
        },
        PrimitiveConstructor::CenterStartEndArc => arc_input(points, request),
    })
}
fn arc_input(points: &[[f64; 2]], request: &AddSketchEntityArgs) -> SketchPrimitiveCreation {
    SketchPrimitiveCreation::Arc {
        center: SketchPointInput::New(points[0]),
        endpoints: [
            SketchPointInput::New(points[1]),
            SketchPointInput::New(points[2]),
        ],
        counter_clockwise: request.ccw,
        construction: request.construction,
    }
}
