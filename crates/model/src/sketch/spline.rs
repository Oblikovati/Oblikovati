// SPDX-License-Identifier: GPL-2.0-only
use super::{SketchEntityId, SketchError};
use oblikovati_api::types::{
    SPLINE_FIT_CHORD, SPLINE_FIT_SMOOTH, SPLINE_FIT_SWEET, SplineFitMethod,
};
mod dimension;
mod owner;
mod storage;
mod variables;
/// Defining points either control the curve or are interpolated by the chosen fit method.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub enum SketchSplineKind {
    Control,
    Fit { method: SplineFitMethod },
}
/// Authored parent identity and shared defining points; no sampled geometry is stored.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchSpline2D {
    id: SketchEntityId,
    points: Vec<SketchEntityId>,
    closed: bool,
    kind: SketchSplineKind,
}
impl SketchSpline2D {
    /// Prepare a parent before ownership resolution; example: `SketchSpline2D::new(id, points, false, SketchSplineKind::Control)?`.
    pub fn new(
        id: SketchEntityId,
        points: Vec<SketchEntityId>,
        closed: bool,
        kind: SketchSplineKind,
    ) -> Result<Self, SketchError> {
        require_spline_definition(id, &points, kind)?;
        Ok(Self {
            id,
            points,
            closed,
            kind,
        })
    }
    /// Inspect the curve identity; example: `spline.id()`.
    pub fn id(&self) -> SketchEntityId {
        self.id
    }
    /// Inspect defining point references in authored order; example: `spline.points()`.
    pub fn points(&self) -> &[SketchEntityId] {
        &self.points
    }
    /// Inspect the closure rule; example: `spline.closed()`.
    pub fn closed(&self) -> bool {
        self.closed
    }
    /// Inspect control or fit intent; example: `spline.kind()`.
    pub fn kind(&self) -> SketchSplineKind {
        self.kind
    }
}
fn require_spline_definition(
    id: SketchEntityId,
    points: &[SketchEntityId],
    kind: SketchSplineKind,
) -> Result<(), SketchError> {
    if id.0 == 0 || points.len() < 2 || points.iter().any(|point| point.0 == 0 || *point == id) {
        return Err(SketchError::Definition(format!(
            "spline {id:?} points {points:?}: expected a nonzero curve ID and at least two nonzero point references distinct from the curve"
        )));
    }
    if let SketchSplineKind::Fit { method } = kind
        && ![SPLINE_FIT_CHORD, SPLINE_FIT_SMOOTH, SPLINE_FIT_SWEET].contains(&method)
    {
        return Err(SketchError::Definition(format!(
            "spline {id:?} fit method {method:?}: expected chord, smooth, or sweet"
        )));
    }
    Ok(())
}
/// Derived offset identity, owned parent reference, and signed working distance (#1874).
#[derive(Clone, Copy, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchOffsetSpline2D {
    id: SketchEntityId,
    parent: SketchEntityId,
    distance: f64,
}
impl SketchOffsetSpline2D {
    /// Prepare an offset without adopting it; example: `SketchOffsetSpline2D::new(id, parent, -2.0)?`.
    pub fn new(
        id: SketchEntityId,
        parent: SketchEntityId,
        distance: f64,
    ) -> Result<Self, SketchError> {
        if id.0 == 0 || parent.0 == 0 || id == parent || !distance.is_finite() {
            return Err(SketchError::Definition(format!(
                "offset spline {id:?} parent {parent:?} distance {distance}: expected distinct nonzero IDs and a finite signed working length"
            )));
        }
        Ok(Self {
            id,
            parent,
            distance,
        })
    }
    /// Inspect the derived curve identity; example: `offset.id()`.
    pub fn id(self) -> SketchEntityId {
        self.id
    }
    /// Inspect the exact named parent; example: `offset.parent()`.
    pub fn parent(self) -> SketchEntityId {
        self.parent
    }
    /// Inspect the signed working distance; example: `offset.distance()`.
    pub fn distance(self) -> f64 {
        self.distance
    }
}

pub(super) fn require_spline_schema(recipe: &super::SketchRecipe) -> Result<(), SketchError> {
    dimension::require_offset_dimension_schema(recipe)?;
    storage::spline_records(recipe).map(|_| ())
}
