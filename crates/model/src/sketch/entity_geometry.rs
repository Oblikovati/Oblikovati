// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchEntityId, SketchError, SketchPoint2D};
use crate::sketch::{SketchArc2D, SketchCircle2D, SketchLine2D, variables::SketchVariable};
use oblikovati_api::types::{
    SKETCH_ENTITY_ARC, SKETCH_ENTITY_CIRCLE, SKETCH_ENTITY_LINE, SKETCH_ENTITY_POINT,
    SketchEntityKind,
};
use std::collections::BTreeMap;
#[derive(Clone, Copy)]
pub(super) enum OwnedSketchGeometry {
    Point(SketchPoint2D),
    Line(SketchLine2D),
    Circle(SketchCircle2D),
    Arc(SketchArc2D),
}
impl OwnedSketchGeometry {
    pub fn id(self) -> SketchEntityId {
        match self {
            Self::Point(shape) => shape.id,
            Self::Line(shape) => shape.id,
            Self::Circle(shape) => shape.id,
            Self::Arc(shape) => shape.id,
        }
    }
    pub fn kind(self) -> SketchEntityKind {
        match self {
            Self::Point(_) => SKETCH_ENTITY_POINT,
            Self::Line(_) => SKETCH_ENTITY_LINE,
            Self::Circle(_) => SKETCH_ENTITY_CIRCLE,
            Self::Arc(_) => SKETCH_ENTITY_ARC,
        }
    }
    pub fn points(self) -> Vec<SketchEntityId> {
        match self {
            Self::Point(shape) => vec![shape.id],
            Self::Line(shape) => shape.endpoints.to_vec(),
            Self::Circle(shape) => vec![shape.center],
            Self::Arc(shape) => vec![shape.center, shape.start, shape.end],
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
        variables
            .into_iter()
            .map(|variable| sketch.variable_column(variable))
            .collect()
    }
}

pub(super) fn catalogue(sketch: &Sketch2D) -> BTreeMap<SketchEntityId, OwnedSketchGeometry> {
    linear_shapes(sketch)
        .chain(circular_shapes(sketch))
        .map(|shape| (shape.id(), shape))
        .collect()
}
fn linear_shapes(sketch: &Sketch2D) -> impl Iterator<Item = OwnedSketchGeometry> + '_ {
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
fn circular_shapes(sketch: &Sketch2D) -> impl Iterator<Item = OwnedSketchGeometry> + '_ {
    sketch
        .circles()
        .iter()
        .copied()
        .map(OwnedSketchGeometry::Circle)
        .chain(sketch.arcs().iter().copied().map(OwnedSketchGeometry::Arc))
}
