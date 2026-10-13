// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchEntityId, SketchError, SketchPoint2D};
use crate::sketch::{
    SketchArc2D, SketchCircle2D, SketchEllipse2D, SketchLine2D, variables::SketchVariable,
};
use oblikovati_api::types::{
    SKETCH_ENTITY_ARC, SKETCH_ENTITY_CIRCLE, SKETCH_ENTITY_CONTROL_POINT_SPLINE,
    SKETCH_ENTITY_ELLIPSE, SKETCH_ENTITY_LINE, SKETCH_ENTITY_OFFSET_SPLINE, SKETCH_ENTITY_POINT,
    SKETCH_ENTITY_SPLINE, SketchEntityKind,
};
use std::collections::BTreeMap;
#[derive(Clone, Copy)]
pub(super) enum OwnedSketchGeometry<'a> {
    Point(SketchPoint2D),
    Line(SketchLine2D),
    Circle(SketchCircle2D),
    Arc(SketchArc2D),
    Ellipse(SketchEllipse2D),
    Spline(&'a super::SketchSpline2D),
    OffsetSpline(super::SketchOffsetSpline2D),
}
impl OwnedSketchGeometry<'_> {
    pub fn ownership_dependencies(self) -> Vec<SketchEntityId> {
        match self {
            Self::Point(_) => Vec::new(),
            Self::OffsetSpline(offset) => vec![offset.parent()],
            _ => self.points(),
        }
    }
    pub fn fit_method(self) -> Option<oblikovati_api::types::SplineFitMethod> {
        if let Self::Spline(parent) = self
            && let super::SketchSplineKind::Fit { method } = parent.kind()
        {
            return Some(method);
        }
        None
    }
    pub fn id(self) -> SketchEntityId {
        match self {
            Self::Point(shape) => shape.id,
            Self::Line(shape) => shape.id,
            Self::Circle(shape) => shape.id,
            Self::Arc(shape) => shape.id,
            Self::Ellipse(shape) => shape.id(),
            Self::Spline(shape) => shape.id(),
            Self::OffsetSpline(shape) => shape.id(),
        }
    }
    pub fn kind(self) -> SketchEntityKind {
        match self {
            Self::Point(_) => SKETCH_ENTITY_POINT,
            Self::Line(_) => SKETCH_ENTITY_LINE,
            Self::Circle(_) => SKETCH_ENTITY_CIRCLE,
            Self::Arc(_) => SKETCH_ENTITY_ARC,
            Self::Ellipse(_) => SKETCH_ENTITY_ELLIPSE,
            Self::Spline(shape) => spline_entity_kind(shape.kind()),
            Self::OffsetSpline(_) => SKETCH_ENTITY_OFFSET_SPLINE,
        }
    }
    pub fn transform_points(
        self,
        transform: super::SketchSelectionTransform,
    ) -> Result<Vec<SketchEntityId>, SketchError> {
        if !matches!(transform, super::SketchSelectionTransform::Move(_))
            && matches!(self, Self::OffsetSpline(_))
        {
            return Err(SketchError::Definition(format!(
                "rotation or reflection entity {:?}: expected an independently owned curve; transform the offset spline parent instead",
                self.id()
            )));
        }
        Ok(self.points())
    }
    pub fn points(self) -> Vec<SketchEntityId> {
        match self {
            Self::Point(shape) => vec![shape.id],
            Self::Line(shape) => shape.endpoints.to_vec(),
            Self::Circle(shape) => vec![shape.center],
            Self::Arc(shape) => vec![shape.center, shape.start, shape.end],
            Self::Ellipse(shape) => vec![shape.center()],
            Self::Spline(shape) => shape.points().to_vec(),
            Self::OffsetSpline(_) => Vec::new(),
        }
    }
    pub fn radius(self, sketch: &Sketch2D) -> Result<f64, SketchError> {
        match self {
            Self::Circle(shape) => Ok(shape.radius),
            Self::Arc(shape) => sketch.arc_radius(shape.id),
            _ => Ok(0.0),
        }
    }
    pub fn columns(self, sketch: &Sketch2D) -> Result<Vec<usize>, SketchError> {
        let mut variables = sketch.point_variables(&self.points());
        if let Self::Circle(shape) = self {
            variables.push(SketchVariable::Radius(shape.id));
        }
        if let Self::Ellipse(shape) = self {
            variables.extend(super::ellipse::variables::ellipse_variables(shape.id()));
        }
        if let Self::OffsetSpline(shape) = self {
            variables.push(SketchVariable::OffsetSpline(shape.id()));
        }
        variables
            .into_iter()
            .map(|variable| sketch.variable_column(variable))
            .collect()
    }
}

pub(super) fn catalogue(sketch: &Sketch2D) -> BTreeMap<SketchEntityId, OwnedSketchGeometry<'_>> {
    linear_shapes(sketch)
        .chain(circular_shapes(sketch))
        .chain(
            sketch
                .ellipses()
                .iter()
                .copied()
                .map(OwnedSketchGeometry::Ellipse),
        )
        .chain(spline_shapes(sketch))
        .map(|shape| (shape.id(), shape))
        .collect()
}
fn linear_shapes(sketch: &Sketch2D) -> impl Iterator<Item = OwnedSketchGeometry<'_>> + '_ {
    sketch
        .points()
        .iter()
        .copied()
        .map(OwnedSketchGeometry::Point)
        .chain(
            sketch
                .lines()
                .iter()
                .copied()
                .map(OwnedSketchGeometry::Line),
        )
}
fn circular_shapes(sketch: &Sketch2D) -> impl Iterator<Item = OwnedSketchGeometry<'_>> + '_ {
    sketch
        .circles()
        .iter()
        .copied()
        .map(OwnedSketchGeometry::Circle)
        .chain(sketch.arcs().iter().copied().map(OwnedSketchGeometry::Arc))
}

fn spline_shapes(sketch: &Sketch2D) -> impl Iterator<Item = OwnedSketchGeometry<'_>> + '_ {
    sketch
        .splines()
        .iter()
        .map(OwnedSketchGeometry::Spline)
        .chain(
            sketch
                .offset_splines()
                .iter()
                .copied()
                .map(OwnedSketchGeometry::OffsetSpline),
        )
}

fn spline_entity_kind(kind: super::SketchSplineKind) -> SketchEntityKind {
    match kind {
        super::SketchSplineKind::Control => SKETCH_ENTITY_CONTROL_POINT_SPLINE,
        super::SketchSplineKind::Fit { .. } => SKETCH_ENTITY_SPLINE,
    }
}
