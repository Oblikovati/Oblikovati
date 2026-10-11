// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{
    SKETCH_ENTITY_ARC, SKETCH_ENTITY_CIRCLE, SKETCH_ENTITY_ELLIPSE, SKETCH_ENTITY_LINE,
    SKETCH_ENTITY_POINT,
};
use oblikovati_model::{
    history::SketchLineCreation,
    quantity::Dimension,
    sketch::{SketchPointInput, SketchPrimitiveCreation},
};
mod points;
pub(super) enum PreparedEntity {
    Primitive(SketchPrimitiveCreation, &'static str),
    InferredLine(SketchLineCreation),
}
#[derive(Clone, Copy)]
enum PrimitiveConstructor {
    FreshTwoPointLine,
    StandalonePoint,
    CenterRadiusCircle,
    AnalyticEllipse,
    CenterStartEndArc,
}
pub(super) fn prepare(
    snapshot: &DocumentSnapshot,
    request: &AddSketchEntityArgs,
) -> Result<PreparedEntity, Error> {
    let constructor = classify_kind(&request.kind)?;
    require_variant(constructor, &request.variant)?;
    let points = points::resolve(snapshot, request)?;
    require_point_count(constructor, &points)?;
    construct_input(constructor, snapshot, &points, request)
}
fn classify_kind(kind: &str) -> Result<PrimitiveConstructor, Error> {
    match kind {
        kind if kind == SKETCH_ENTITY_LINE.0 => Ok(PrimitiveConstructor::FreshTwoPointLine),
        kind if kind == SKETCH_ENTITY_POINT.0 => Ok(PrimitiveConstructor::StandalonePoint),
        kind if kind == SKETCH_ENTITY_CIRCLE.0 => Ok(PrimitiveConstructor::CenterRadiusCircle),
        kind if kind == SKETCH_ENTITY_ELLIPSE.0 => Ok(PrimitiveConstructor::AnalyticEllipse),
        kind if kind == SKETCH_ENTITY_ARC.0 => Ok(PrimitiveConstructor::CenterStartEndArc),
        _ => Err(Error(format!(
            "sketch.addEntity kind {kind:?}: expected a ported line, point, circle, arc, or ellipse constructor; other kinds are not ported"
        ))),
    }
}
fn require_variant(constructor: PrimitiveConstructor, variant: &str) -> Result<(), Error> {
    // The Go line and ellipse constructors ignore variant (aa8297d1, #3775).
    if matches!(
        constructor,
        PrimitiveConstructor::FreshTwoPointLine | PrimitiveConstructor::AnalyticEllipse
    ) {
        return Ok(());
    }
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
    let expected = match constructor {
        PrimitiveConstructor::FreshTwoPointLine => 2,
        PrimitiveConstructor::CenterStartEndArc => 3,
        _ => 1,
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
            Self::FreshTwoPointLine => SKETCH_ENTITY_LINE.0.as_ref(),
            Self::StandalonePoint => SKETCH_ENTITY_POINT.0.as_ref(),
            Self::AnalyticEllipse => SKETCH_ENTITY_ELLIPSE.0.as_ref(),
            Self::CenterRadiusCircle => SKETCH_ENTITY_CIRCLE.0.as_ref(),
            Self::CenterStartEndArc => SKETCH_ENTITY_ARC.0.as_ref(),
        }
    }
    fn variant(self) -> &'static str {
        match self {
            Self::FreshTwoPointLine => "",
            Self::StandalonePoint | Self::AnalyticEllipse => "",
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
) -> Result<PreparedEntity, Error> {
    let input = match constructor {
        PrimitiveConstructor::FreshTwoPointLine => return Ok(inferred_line_input(points, request)),
        PrimitiveConstructor::StandalonePoint => SketchPrimitiveCreation::Point {
            position: points[0],
        },
        PrimitiveConstructor::AnalyticEllipse => ellipse_input(snapshot, points[0], request)?,
        PrimitiveConstructor::CenterRadiusCircle => circle_input(snapshot, points[0], request)?,
        PrimitiveConstructor::CenterStartEndArc => arc_input(points, request),
    };
    Ok(PreparedEntity::Primitive(input, constructor.kind()))
}
fn inferred_line_input(points: &[[f64; 2]], request: &AddSketchEntityArgs) -> PreparedEntity {
    PreparedEntity::InferredLine(SketchLineCreation {
        endpoints: [points[0], points[1]],
        construction: request.construction,
        centerline: request.centerline,
    })
}
fn circle_input(
    snapshot: &DocumentSnapshot,
    center: [f64; 2],
    request: &AddSketchEntityArgs,
) -> Result<SketchPrimitiveCreation, Error> {
    Ok(SketchPrimitiveCreation::Circle {
        center: SketchPointInput::New(center),
        radius: length_expression(snapshot, &request.radius)?,
        construction: request.construction,
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

fn ellipse_input(
    snapshot: &DocumentSnapshot,
    center: [f64; 2],
    request: &AddSketchEntityArgs,
) -> Result<SketchPrimitiveCreation, Error> {
    let axis = request.axis.as_deref().unwrap_or_default();
    let major_axis = <[f64; 2]>::try_from(axis).map_err(|_| {
        Error(format!(
            "sketch.addEntity ellipse axis {axis:?}: expected two finite direction components"
        ))
    })?;
    Ok(SketchPrimitiveCreation::Ellipse {
        center: SketchPointInput::New(center),
        major_axis,
        radii: [
            length_expression(snapshot, &request.major_radius)?,
            length_expression(snapshot, &request.minor_radius)?,
        ],
        construction: request.construction,
    })
}
