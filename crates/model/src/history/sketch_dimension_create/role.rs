// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::SketchPoint2D;
use oblikovati_api::types::{OVER_CONSTRAINED_APPLY_DRIVING, OVER_CONSTRAINED_PROMPT};

pub(super) enum DimensionRolePolicy {
    Explicit,
    Placement,
}
pub(super) enum DimensionRoleProbe {
    Explicit,
    Redundancy {
        prior: usize,
        anchors: Vec<SketchPoint2D>,
        prompt: bool,
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
        datums: AuthoringDatums<'_>,
    ) -> Result<SketchDimensionCreationReport, SketchDimensionCreationError> {
        self.create_dimension_with_role(
            sketch,
            input,
            options,
            ids,
            DimensionRolePolicy::Placement,
            datums,
        )
    }
    pub(super) fn prepare_dimension_role(
        &self,
        sketch: DocumentSketchId,
        geometry: &Sketch2D,
        input: &SketchDimensionCreation,
        policy: DimensionRolePolicy,
        options: SolveOptions,
        datums: AuthoringDatums<'_>,
    ) -> Result<DimensionRoleProbe, SketchError> {
        let behavior = self.effective_sketch_settings().over_constrained_behavior;
        if input.reference
            || behavior == OVER_CONSTRAINED_APPLY_DRIVING
            || (matches!(policy, DimensionRolePolicy::Explicit)
                && behavior != OVER_CONSTRAINED_PROMPT)
        {
            return Ok(DimensionRoleProbe::Explicit);
        }
        let anchors = self.sketch_authoring_anchors(sketch, datums)?;
        let prior = geometry
            .analyze_with_grounded_points(&self.parameters, &anchors, options.rank_tolerance)?
            .redundant_equations;
        Ok(DimensionRoleProbe::Redundancy {
            prior,
            anchors,
            prompt: behavior == OVER_CONSTRAINED_PROMPT,
        })
    }
    pub(super) fn classify_dimension_role(
        &mut self,
        sketch: DocumentSketchId,
        id: SketchConstraintId,
        probe: DimensionRoleProbe,
        options: SolveOptions,
    ) -> Result<(), SketchDimensionCreationError> {
        let DimensionRoleProbe::Redundancy {
            prior,
            anchors,
            prompt,
        } = probe
        else {
            return Ok(());
        };
        let geometry = self.sketches.get(sketch)?.geometry();
        // #3824: compare the same prepared seed and references; choose the final role before the solve.
        let after = geometry
            .analyze_with_grounded_points(&self.parameters, &anchors, options.rank_tolerance)?
            .redundant_equations;
        if after > prior {
            if prompt {
                return Err(SketchDimensionCreationError::ChoiceRequired);
            }
            self.sketches.set_dimension_reference(sketch, id, true)?;
        }
        Ok(())
    }
}
