// SPDX-License-Identifier: GPL-2.0-only
use super::DocumentSnapshot;
use crate::{
    document_sketch::DocumentSketchId,
    sketch::{Sketch2D, SketchError},
};
use oblikovati_solve::nonlinear::{SolveOptions, SolveTermination};
impl DocumentSnapshot {
    pub(super) fn current_sketch_authoring_geometry(
        &self,
        sketch: DocumentSketchId,
        options: SolveOptions,
    ) -> Result<Sketch2D, SketchError> {
        self.current_sketch_authoring_geometry_from(sketch, options, None)
    }
    pub(super) fn current_sketch_authoring_geometry_from(
        &self,
        sketch: DocumentSketchId,
        options: SolveOptions,
        datums: Option<&crate::feature::work_geometry::WorkGeometryEvaluation>,
    ) -> Result<Sketch2D, SketchError> {
        let geometry = self.sketches.get(sketch)?.geometry();
        let anchors = self.sketch_authoring_anchors(sketch, datums)?;
        let solved =
            geometry.solve_with_grounded_points(&self.parameters, &anchors, options, || false)?;
        let termination = solved.numerical().termination;
        if termination != SolveTermination::Converged {
            return Err(SketchError::Definition(format!(
                "sketch {} authoring source solve {termination:?}: expected converged current coordinates; free variables are permitted",
                sketch.0
            )));
        }
        Ok(solved.into_candidate())
    }
    pub(super) fn sketch_authoring_anchors(
        &self,
        sketch: DocumentSketchId,
        datums: Option<&crate::feature::work_geometry::WorkGeometryEvaluation>,
    ) -> Result<Vec<crate::sketch::SketchPoint2D>, SketchError> {
        match datums {
            Some(datums) => self.resolve_sketch_projected_points_with_datums(sketch, datums),
            None => self.resolve_sketch_projected_points(sketch),
        }
    }
}
