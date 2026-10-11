// SPDX-License-Identifier: GPL-2.0-only
use super::DocumentSnapshot;
use crate::sketch::SketchError;
use oblikovati_api::types::MODEL_PARAM;
use std::collections::BTreeSet;
impl DocumentSnapshot {
    /// Retire only Model sources whose final dimension or feature scalar owner was removed.
    /// Example: `candidate.remove_retired_dimension_sources([parameter].into())?;`.
    pub(super) fn remove_retired_dimension_sources(
        &mut self,
        sources: BTreeSet<u64>,
    ) -> Result<(), SketchError> {
        for parameter in sources {
            if self.retains_dimension_source(parameter) || !self.is_model_source(parameter) {
                continue;
            }
            self.parameters.validate_owned_scalar(parameter)
                .and_then(|()| self.parameters.remove_for_owner(parameter))
                .map_err(|cause| SketchError::Definition(format!(
                    "deleted dimension parameter {parameter}: expected a removable Model scalar: {cause}"
                )))?;
        }
        Ok(())
    }

    fn is_model_source(&self, parameter: u64) -> bool {
        self.parameters
            .get(parameter)
            .is_some_and(|source| source.kind == MODEL_PARAM)
    }

    fn retains_dimension_source(&self, parameter: u64) -> bool {
        // Reference dimensions still own their source even though they do not drive solver rows.
        self.sketches.sketches().iter().any(|placed| {
            placed
                .geometry()
                .dimensional_constraints()
                .any(|(_, source)| source == parameter)
        }) || self.work_geometry.features().iter().any(|feature| {
            feature
                .owned_scalars()
                .iter()
                .any(|binding| binding.parameter == parameter)
        })
    }
}
