// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::{Error, types::*, wire::AddDimensionArgs};
use oblikovati_model::{
    history::SketchDimensionCreation,
    sketch::{
        DistanceMeasure, MeasuredDimensionOperands, SeededDimensionOperands, Sketch2D,
        SketchDimensionalConstraintCreation, SketchEntityId, TangentDistanceOptions,
    },
};
mod operands;
use SketchDimensionalConstraintCreation::{Measured, Seeded};
use operands::{Shape, require};
pub(super) fn prepare(
    geometry: &Sketch2D,
    scale: f64,
    request: &AddDimensionArgs,
) -> Result<SketchDimensionCreation, Error> {
    let measure = orientation(&request.orientation)?;
    let operands = constructor(geometry, request, measure)?;
    let text_point = annotation(request, scale)?;
    Ok(SketchDimensionCreation {
        operands,
        text_point,
        expression: request.expression.clone(),
        reference: request.driven,
    })
}
fn annotation(request: &AddDimensionArgs, scale: f64) -> Result<Option<[f64; 2]>, Error> {
    let point = request.text_point.as_deref().unwrap_or_default();
    if point.is_empty() {
        return Ok(None);
    }
    super::super::coordinates::working_pair(point, scale, "dimension textPoint").map(Some)
}
fn orientation(name: &str) -> Result<DistanceMeasure, Error> {
    match name {
        "" | "aligned" => Ok(DistanceMeasure::Aligned),
        "horizontal" => Ok(DistanceMeasure::Horizontal),
        "vertical" => Ok(DistanceMeasure::Vertical),
        _ => Err(Error(format!(
            "dimension orientation {name:?}: expected aligned, horizontal, vertical, or empty"
        ))),
    }
}
fn constructor(
    geometry: &Sketch2D,
    request: &AddDimensionArgs,
    measure: DistanceMeasure,
) -> Result<SketchDimensionalConstraintCreation, Error> {
    let kind = DimensionConstraintKind(request.kind.clone().into());
    match kind {
        kind if kind == DIM_CONSTRAINT_DISTANCE => distance(geometry, request, measure),
        kind if kind == DIM_CONSTRAINT_RADIUS || kind == DIM_CONSTRAINT_DIAMETER => {
            radial(geometry, request, kind == DIM_CONSTRAINT_DIAMETER)
        }
        kind if is_ellipse_radius(&kind) => ellipse_radius(geometry, request, kind),
        kind if kind == DIM_CONSTRAINT_ARC_LENGTH => arc_length(geometry, request),
        kind if kind == DIM_CONSTRAINT_ANGLE => angle(geometry, request),
        kind if kind == DIM_CONSTRAINT_THREE_POINT_ANGLE => three_point_angle(geometry, request),
        kind if kind == DIM_CONSTRAINT_OFFSET => offset(geometry, request),
        kind if kind == DIM_CONSTRAINT_TANGENT_DISTANCE => tangent(geometry, request),
        _ => unsupported_kind(&kind),
    }
}
fn is_ellipse_radius(kind: &DimensionConstraintKind) -> bool {
    *kind == DIM_CONSTRAINT_ELLIPSE_RADIUS || *kind == DIM_CONSTRAINT_ELLIPSE_MINOR_RADIUS
}
fn unsupported_kind(
    kind: &DimensionConstraintKind,
) -> Result<SketchDimensionalConstraintCreation, Error> {
    let expected = if *kind == DIM_CONSTRAINT_OFFSET_SPLINE {
        "native offset-spline geometry; this R3 capability is not ported"
    } else {
        "a supported canonical dimension kind"
    };
    Err(Error(format!(
        "dimension constructor {kind:?}: expected {expected}"
    )))
}

fn distance(
    geometry: &Sketch2D,
    request: &AddDimensionArgs,
    measure: DistanceMeasure,
) -> Result<SketchDimensionalConstraintCreation, Error> {
    let points = require(geometry, request, [Shape::Point; 2])?;
    Ok(Measured(MeasuredDimensionOperands::Distance {
        points,
        measure,
    }))
}
fn radial(
    geometry: &Sketch2D,
    request: &AddDimensionArgs,
    diameter: bool,
) -> Result<SketchDimensionalConstraintCreation, Error> {
    let [curve] = require(geometry, request, [Shape::Circular])?;
    let input = if diameter {
        MeasuredDimensionOperands::Diameter { curve }
    } else {
        MeasuredDimensionOperands::Radius { curve }
    };
    Ok(Measured(input))
}
fn arc_length(
    geometry: &Sketch2D,
    request: &AddDimensionArgs,
) -> Result<SketchDimensionalConstraintCreation, Error> {
    let [arc] = require(geometry, request, [Shape::Arc])?;
    Ok(Measured(MeasuredDimensionOperands::ArcLength { arc }))
}
fn angle(
    geometry: &Sketch2D,
    request: &AddDimensionArgs,
) -> Result<SketchDimensionalConstraintCreation, Error> {
    let lines = require(geometry, request, [Shape::Line; 2])?;
    Ok(Seeded(SeededDimensionOperands::Angle(lines)))
}
fn three_point_angle(
    geometry: &Sketch2D,
    request: &AddDimensionArgs,
) -> Result<SketchDimensionalConstraintCreation, Error> {
    let points = require(geometry, request, [Shape::Point; 3])?;
    Ok(Seeded(SeededDimensionOperands::ThreePointAngle(points)))
}
fn offset(
    geometry: &Sketch2D,
    request: &AddDimensionArgs,
) -> Result<SketchDimensionalConstraintCreation, Error> {
    let point_line = require(geometry, request, [Shape::Point, Shape::Line])?;
    Ok(Seeded(SeededDimensionOperands::Offset {
        point_line,
        linear_diameter: request.linear_diameter,
    }))
}
fn tangent(
    geometry: &Sketch2D,
    request: &AddDimensionArgs,
) -> Result<SketchDimensionalConstraintCreation, Error> {
    let line_curve = require(geometry, request, [Shape::Line, Shape::Circular])?;
    let options = TangentDistanceOptions {
        far_side: request.far_side,
        linear_diameter: request.linear_diameter,
    };
    Ok(Seeded(SeededDimensionOperands::TangentDistance {
        line_curve,
        options,
    }))
}

fn ellipse_radius(
    geometry: &Sketch2D,
    request: &AddDimensionArgs,
    kind: DimensionConstraintKind,
) -> Result<SketchDimensionalConstraintCreation, Error> {
    let [ellipse] = require(geometry, request, [Shape::Ellipse])?;
    let operands = if kind == DIM_CONSTRAINT_ELLIPSE_MINOR_RADIUS {
        MeasuredDimensionOperands::EllipseMinorRadius { ellipse }
    } else {
        MeasuredDimensionOperands::EllipseRadius { ellipse }
    };
    Ok(Measured(operands))
}
