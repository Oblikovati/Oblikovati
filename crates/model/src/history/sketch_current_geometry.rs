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
        let geometry = self.sketches.get(sketch)?.geometry();
        let anchors = self.resolve_sketch_projected_points(sketch)?;
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
}
