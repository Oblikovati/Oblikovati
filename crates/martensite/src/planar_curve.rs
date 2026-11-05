// SPDX-License-Identifier: GPL-2.0-only
//! Application-owned analytic inputs; no mutable kernel handle escapes a call.
use crate::PlanarSplineDefinition;
mod circumcircle;
mod sketch_construction;
pub use sketch_construction::MartensiteSketchConstruction;
mod construct;
pub use circumcircle::{CircumcircleOptions, PlanarCircumcircle, construct_planar_circumcircle};
mod failure;
mod report;
/// Owned sketch curve conversion and native incidence queries.
pub mod sketch;
mod transition;
pub use failure::{PlanarCurveFailure, PlanarCurveFailureKind};
pub use transition::{CurvePosition, CurveTransition, TouchSide, TransitionKind};

/// Conic center, local X axis, and positive-parameter travel direction.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct PlanarConicFrame {
    pub center: [f64; 2],
    pub axis: [f64; 2],
    pub counterclockwise: bool,
}
/// Native parameters use unit length for lines and radians for conics.
#[derive(Clone, Debug, PartialEq)]
pub enum PlanarGeometry {
    Line {
        origin: [f64; 2],
        direction: [f64; 2],
    },
    Circle {
        frame: PlanarConicFrame,
        radius: f64,
    },
    Ellipse {
        frame: PlanarConicFrame,
        radii: [f64; 2],
    },
    Spline(PlanarSplineDefinition),
}
/// Increasing native bounds stay within a nonperiodic basis or span at most one period.
/// Offsets are left of forward travel; periodic bounds keep their supplied phase.
#[derive(Clone, Debug, PartialEq)]
pub struct PlanarCurveDefinition {
    pub geometry: PlanarGeometry,
    pub bounds: Option<[f64; 2]>,
    pub left_offset: f64,
}
/// Explicit native solver tolerances and a separate spatial result acceptance bound.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct PlanarIntersectionOptions {
    pub confusion_tolerance: f64,
    pub intersection_tolerance: f64,
    pub maximum_point_residual: f64,
}
/// One incidence on both operands, including measured distances to the reported point.
#[derive(Clone, Debug, PartialEq)]
pub struct PlanarIntersectionPoint {
    pub position: [f64; 2],
    pub parameters: [f64; 2],
    pub residuals: [f64; 2],
    pub transitions: [CurveTransition; 2],
}
/// Absent endpoints denote unbounded overlap; orientation refers to the two operands.
#[derive(Clone, Debug, PartialEq)]
pub struct PlanarOverlap {
    pub first: Option<PlanarIntersectionPoint>,
    pub last: Option<PlanarIntersectionPoint>,
    pub opposite: bool,
}
/// Completed native results; completion is not a certificate of root completeness.
#[derive(Clone, Debug, PartialEq)]
pub struct PlanarIntersectionReport {
    pub points: Vec<PlanarIntersectionPoint>,
    pub overlaps: Vec<PlanarOverlap>,
}
/// Intersect fresh analytic curves and check every finite reported incidence.
/// Example: `let report = intersect_planar_curves(&first, &second, options)?;`.
pub fn intersect_planar_curves(
    first: &PlanarCurveDefinition,
    second: &PlanarCurveDefinition,
    options: PlanarIntersectionOptions,
) -> Result<PlanarIntersectionReport, PlanarCurveFailure> {
    construct::require_options(options)?;
    let first_curve =
        construct::build_curve(first).map_err(|failure| failure.with_input("first", first))?;
    let second_curve =
        construct::build_curve(second).map_err(|failure| failure.with_input("second", second))?;
    report::intersect(&first_curve, &second_curve, options).map_err(|failure| {
        failure
            .with_input("first", first)
            .with_input("second", second)
    })
}
