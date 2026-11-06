// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::SketchPoint2D;
use oblikovati_api::types::OVER_CONSTRAINED_APPLY_DRIVING;

pub(super) enum DimensionRolePolicy {
    Explicit,
    Placement,
}
pub(super) enum DimensionRoleProbe {
    Explicit,
    Redundancy {
        prior: usize,
        anchors: Vec<SketchPoint2D>,
    },
}
impl DocumentSnapshot {
    /// Classify a placement lock's role before solving its candidate.
    /// Example: `snapshot.create_placement_dimension(sketch,&input,options,ids)?` inside a placement edit.
    pub(crate) fn create_placement_dimension(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchDimensionCreation,
        options: SolveOptions,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<SketchDimensionCreationReport, SketchDimensionCreationError> {
        self.create_dimension_with_role(sketch, input, options, ids, DimensionRolePolicy::Placement)
    }
    pub(super) fn prepare_dimension_role(
        &self,
        sketch: DocumentSketchId,
        geometry: &Sketch2D,
        input: &SketchDimensionCreation,
        policy: DimensionRolePolicy,
        options: SolveOptions,
    ) -> Result<DimensionRoleProbe, SketchError> {
        if matches!(policy, DimensionRolePolicy::Explicit)
            || input.reference
            || self.effective_sketch_settings().over_constrained_behavior
                == OVER_CONSTRAINED_APPLY_DRIVING
        {
            return Ok(DimensionRoleProbe::Explicit);
        }
        let anchors = self.resolve_sketch_projected_points(sketch)?;
        let prior = geometry
            .analyze_with_grounded_points(&self.parameters, &anchors, options.rank_tolerance)?
            .redundant_equations;
        Ok(DimensionRoleProbe::Redundancy { prior, anchors })
    }
    pub(super) fn classify_dimension_role(
        &mut self,
        sketch: DocumentSketchId,
        id: SketchConstraintId,
        probe: DimensionRoleProbe,
        options: SolveOptions,
    ) -> Result<(), SketchError> {
        let DimensionRoleProbe::Redundancy { prior, anchors } = probe else {
            return Ok(());
        };
        let geometry = self.sketches.get(sketch)?.geometry();
        // #3824: compare the same prepared seed and references; choose the final role before the solve.
        let after = geometry
            .analyze_with_grounded_points(&self.parameters, &anchors, options.rank_tolerance)?
            .redundant_equations;
        if after > prior {
            self.sketches.set_dimension_reference(sketch, id, true)?;
        }
        Ok(())
    }
}
