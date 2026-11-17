// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSketchId, DocumentSketches};
use crate::{
    parameter::ParameterCollection,
    sketch::{SketchError, SketchPoint2D},
};
use oblikovati_solve::nonlinear::SolveOptions;
impl DocumentSketches {
    /// Recompute the corner component while retaining the full sketch's existing health evidence.
    /// Example: `sketches.recompute_corner_component(id, owner, &parameters, &anchors, options)?`.
    pub fn recompute_corner_component(
        &mut self,
        sketch: DocumentSketchId,
        owner: crate::sketch::SketchEntityId,
        parameters: &ParameterCollection<u64>,
        anchors: &[SketchPoint2D],
        options: SolveOptions,
    ) -> Result<(), SketchError> {
        let index = self.index(sketch)?;
        let solved = self.sketches[index]
            .geometry
            .solve_corner_component(owner, parameters, anchors, options)?;
        self.require_unique_entities(Some(sketch), &solved)?;
        super::point_projection::validate_bindings(
            &solved,
            &self.sketches[index].point_projections,
        )?;
        self.sketches[index].geometry = solved;
        Ok(())
    }
}
