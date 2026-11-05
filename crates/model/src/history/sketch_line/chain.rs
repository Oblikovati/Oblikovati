// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{
    sketch::{
        SketchEntityId,
        chain_layout::{line_chain_indices, require_chain_vertices},
    },
    sketch_settings::SketchSettings,
};

/// Continuous Line tool vertices; a closed two-point chain remains one open segment (Go #2024).
#[derive(Clone, Debug, PartialEq)]
pub struct SketchLineChainCreation {
    pub points: Vec<[f64; 2]>,
    pub closed: bool,
    pub construction: bool,
    pub centerline: bool,
}
/// Visible vertices and ordered line reports preserve shared identity and per-segment inference.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchLineChainCreationReport {
    pub sketch: DocumentSketchId,
    pub points: Vec<SketchEntityId>,
    pub lines: Vec<SketchLineCreationReport>,
}
impl DocumentSnapshot {
    /// Author the continuous Line tool's visible chain atomically against current solved geometry.
    /// Example: `snapshot.create_inferred_sketch_line_chain(sketch, &input, policy, options, ids)?;`.
    pub fn create_inferred_sketch_line_chain(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchLineChainCreation,
        policy: SketchInferencePolicy,
        solve_options: SolveOptions,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<SketchLineChainCreationReport, SketchLineCreationError> {
        require_chain_vertices(&input.points, false).map_err(SketchLineCreationError::Sketch)?;
        policy
            .require_valid()
            .map_err(SketchLineCreationError::Sketch)?;
        let mut geometry = self
            .current_sketch_authoring_geometry(sketch, solve_options)
            .map_err(SketchLineCreationError::Sketch)?;
        let report = input.append_chain(
            &mut geometry,
            sketch,
            &self.effective_sketch_settings(),
            policy,
            ids,
        )?;
        self.sketches
            .replace_geometry(sketch, geometry)
            .map_err(SketchLineCreationError::Sketch)?;
        Ok(report)
    }
}
impl SketchLineChainCreation {
    fn append_chain(
        &self,
        geometry: &mut Sketch2D,
        sketch: DocumentSketchId,
        settings: &SketchSettings,
        policy: SketchInferencePolicy,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<SketchLineChainCreationReport, SketchLineCreationError> {
        let first = append_visible_vertex(geometry, sketch, self.points[0], ids)?;
        let mut report = SketchLineChainCreationReport {
            sketch,
            points: vec![first],
            lines: Vec::new(),
        };
        let closed = self.closed && self.points.len() >= 3;
        for indices in line_chain_indices(self.points.len(), closed) {
            if indices[1] == report.points.len() {
                report.points.push(append_visible_vertex(
                    geometry,
                    sketch,
                    self.points[indices[1]],
                    ids,
                )?);
            }
            let endpoints = indices.map(|index| report.points[index]);
            report
                .lines
                .push(self.append_segment(geometry, sketch, endpoints, settings, policy, ids)?);
        }
        Ok(report)
    }
    fn append_segment(
        &self,
        geometry: &mut Sketch2D,
        sketch: DocumentSketchId,
        points: [SketchEntityId; 2],
        settings: &SketchSettings,
        policy: SketchInferencePolicy,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<SketchLineCreationReport, SketchLineCreationError> {
        let inference = geometry
            .infer_owned_line_placement(points, settings, policy)
            .map_err(SketchLineCreationError::Sketch)?;
        let creation = SketchPrimitiveCreation::Line {
            endpoints: points.map(SketchPointInput::Owned),
            construction: self.construction,
            centerline: self.centerline,
        };
        append_inferred_primitive(
            geometry,
            sketch,
            &creation,
            inference,
            settings.auto_apply_constraints,
            ids,
        )
    }
}
fn append_visible_vertex(
    geometry: &mut Sketch2D,
    sketch: DocumentSketchId,
    position: [f64; 2],
    ids: &mut RecipeReservations<'_>,
) -> Result<SketchEntityId, SketchLineCreationError> {
    let primitive = SketchPrimitiveCreation::Point { position };
    append_primitive(geometry, sketch, &primitive, ids)
        .map(|report| report.entity)
        .map_err(SketchLineCreationError::Primitive)
}
