// SPDX-License-Identifier: GPL-2.0-only

use super::result::WorkGeometryResult;
use super::{WorkGeometryDefinition as WorkPlaneDefinition, WorkGeometryError as WorkPlaneError};
mod queries;
use crate::{
    feature::{
        FeatureHealthRecord, FeatureId, FeatureReplayController, FeatureReplayError,
        FeatureReplayRange, FeatureReplayStep, FeatureSequence,
    },
    parameter::ParameterCollection,
};
use std::collections::BTreeMap;

/// Current validated construction frames plus complete replay health, without stale fallback.
#[derive(Debug, Default)]
pub struct WorkGeometryEvaluation {
    datums: BTreeMap<FeatureId, WorkGeometryResult>,
    pub health: Vec<FeatureHealthRecord<WorkPlaneError>>,
}

impl FeatureSequence<WorkPlaneDefinition> {
    /// Evaluate current construction frames in recipe order; example: `sequence.evaluate_work_planes(&parameters)?`.
    pub fn evaluate_work_geometry(
        &self,
        parameters: &ParameterCollection<u64>,
    ) -> Result<WorkGeometryEvaluation, FeatureReplayError> {
        let range = FeatureReplayRange {
            start: 0,
            end: self.effective_end(),
        };
        self.validate_work_plane_reads()?;
        let mut controller =
            FeatureReplayController::new_with_parameters(self, range, vec![], parameters)?;
        let mut evaluation = WorkGeometryEvaluation::default();
        evaluation.replay(self, parameters, &mut controller)?;
        evaluation.health = controller.finish()?;
        Ok(evaluation)
    }

    pub(crate) fn validate_work_plane_reads(&self) -> Result<(), FeatureReplayError> {
        for feature in self.features() {
            let reads = feature.definition().reads();
            let missing: Vec<_> = reads
                .keys()
                .difference(feature.reads().keys())
                .copied()
                .collect();
            if !missing.is_empty() {
                return Err(FeatureReplayError::Footprint {
                    feature: feature.id(),
                    missing,
                });
            }
        }
        Ok(())
    }
}

impl WorkGeometryEvaluation {
    fn replay(
        &mut self,
        sequence: &FeatureSequence<WorkPlaneDefinition>,
        parameters: &ParameterCollection<u64>,
        controller: &mut FeatureReplayController<'_, WorkPlaneDefinition, WorkPlaneError>,
    ) -> Result<(), FeatureReplayError> {
        while let Some(step) = controller.advance()? {
            if let FeatureReplayStep::Evaluate(id) = step {
                let outcome = self.evaluate_entry(sequence, id, parameters);
                controller.record_result(id, outcome)?;
            }
        }
        Ok(())
    }
    fn evaluate_entry(
        &mut self,
        sequence: &FeatureSequence<WorkPlaneDefinition>,
        id: FeatureId,
        parameters: &ParameterCollection<u64>,
    ) -> Result<(), WorkPlaneError> {
        let feature = sequence.get(id).map_err(|_| WorkPlaneError::Unavailable {
            feature: id,
            expected: "datum",
        })?;
        feature.validate_plane_metadata()?;
        if matches!(feature.definition(), WorkPlaneDefinition::SketchPrism(_)) {
            return Err(WorkPlaneError::Kind(super::WorkGeometryKindMismatch {
                feature: id,
                expected: "datum",
                actual: "sketch prism input",
            }));
        }
        let datum = WorkGeometryResult::evaluate(feature.definition().clone(), parameters, self)?;
        let datum = datum.with_plane_orientation(feature)?;
        self.datums.insert(id, datum);
        Ok(())
    }
}
