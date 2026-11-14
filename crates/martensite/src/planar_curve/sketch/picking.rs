// SPDX-License-Identifier: GPL-2.0-only
use super::{SketchCurveFailure, SketchLineDomain, shape};
use crate::planar_curve::{
    PlanarPickOptionsV1, PlanarPickPoint, closest, nearest_planar_curve_candidate,
};
use oblikovati_model::sketch::{OwnedSketchGeometry, Sketch2D, SketchEntityId, SketchReadIndex};

/// Authored model identity plus a checked analytic witness in sketch coordinates.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchEntityPickWitness {
    pub entity: SketchEntityId,
    pub position: [f64; 2],
    pub distance: f64,
    pub curve: Option<PlanarPickPoint>,
}
/// Query a model point or finite curve without reading display samples or changing geometry.
/// Example: `nearest_sketch_entity_candidate(&sketch, entity, cursor, options)?;`.
pub fn nearest_sketch_entity_candidate(
    sketch: &Sketch2D,
    entity: SketchEntityId,
    query: [f64; 2],
    options: PlanarPickOptionsV1,
) -> Result<SketchEntityPickWitness, SketchCurveFailure> {
    SketchPickQuery::new(sketch, query, options)
        .map_err(|cause| SketchCurveFailure::curve(entity, cause))?
        .candidate(entity)
}
fn nearest_index_candidate(
    index: &SketchReadIndex<'_>,
    entity: SketchEntityId,
    query: [f64; 2],
    options: PlanarPickOptionsV1,
) -> Result<SketchEntityPickWitness, SketchCurveFailure> {
    let geometry = index
        .entity(entity)
        .map_err(|cause| SketchCurveFailure::model(entity, cause))?;
    if let OwnedSketchGeometry::Point(point) = geometry {
        return point_witness(entity, point.position, query);
    }
    curve_witness(index, entity, query, options)
}
fn curve_witness(
    index: &SketchReadIndex<'_>,
    entity: SketchEntityId,
    query: [f64; 2],
    options: PlanarPickOptionsV1,
) -> Result<SketchEntityPickWitness, SketchCurveFailure> {
    let resolved = shape::resolve(
        index,
        entity,
        SketchLineDomain::Segment,
        options.maximum_position_residual,
    )?;
    let curve = nearest_planar_curve_candidate(&resolved.definition, query, options)
        .map_err(|cause| SketchCurveFailure::curve(entity, cause))?;
    Ok(SketchEntityPickWitness {
        entity,
        position: curve.position,
        distance: curve.distance,
        curve: Some(curve),
    })
}
fn point_witness(
    entity: SketchEntityId,
    position: [f64; 2],
    query: [f64; 2],
) -> Result<SketchEntityPickWitness, SketchCurveFailure> {
    let distance = (position[0] - query[0]).hypot(position[1] - query[1]);
    if !distance.is_finite() {
        return Err(SketchCurveFailure::input(
            entity,
            format!("point {position:?} and cursor {query:?}: expected a finite working distance"),
        ));
    }
    Ok(SketchEntityPickWitness {
        entity,
        position,
        distance,
        curve: None,
    })
}

/// One immutable geometry index for all candidates under the same cursor and policy.
pub struct SketchPickQuery<'a> {
    index: SketchReadIndex<'a>,
    query: [f64; 2],
    options: PlanarPickOptionsV1,
}
impl<'a> SketchPickQuery<'a> {
    /// Validate the request once and borrow its geometry for the complete pick pass.
    /// Example: `let query = SketchPickQuery::new(&sketch, cursor, options)?;`.
    pub fn new(
        sketch: &'a Sketch2D,
        query: [f64; 2],
        options: PlanarPickOptionsV1,
    ) -> Result<Self, crate::planar_curve::PlanarCurveFailure> {
        closest::require_pick_options(query, options)?;
        Ok(Self {
            index: sketch.read_index(),
            query,
            options,
        })
    }
    /// Resolve one current model identity with the shared index and numeric policy.
    /// Example: `query.candidate(line_id)?` for a finite segment hit witness.
    pub fn candidate(
        &self,
        entity: SketchEntityId,
    ) -> Result<SketchEntityPickWitness, SketchCurveFailure> {
        nearest_index_candidate(&self.index, entity, self.query, self.options)
    }
}
