// SPDX-License-Identifier: GPL-2.0-only

use super::{WorkPlaneDefinition, WorkPlaneError, WorkPlaneResolver};
use crate::{
    feature::{
        FeatureHealthRecord, FeatureId, FeatureReplayController, FeatureReplayError,
        FeatureReplayRange, FeatureReplayStep, FeatureSequence,
    },
    parameter::ParameterCollection,
    sketch::SketchPlane,
};
use std::collections::BTreeMap;

/// Current validated construction frames plus complete replay health, without stale fallback.
#[derive(Debug, Default)]
pub struct WorkPlaneEvaluation {
    planes: BTreeMap<FeatureId, SketchPlane>,
    pub health: Vec<FeatureHealthRecord<WorkPlaneError>>,
}

impl WorkPlaneResolver for WorkPlaneEvaluation {
    fn plane(&self, feature: FeatureId) -> Result<SketchPlane, WorkPlaneError> {
        self.planes
            .get(&feature)
            .copied()
            .ok_or(WorkPlaneError::Unavailable(feature))
    }
}

impl FeatureSequence<WorkPlaneDefinition> {
    /// Evaluate current construction frames in recipe order; example: `sequence.evaluate_work_planes(&parameters)?`.
    pub fn evaluate_work_planes(
        &self,
        parameters: &ParameterCollection<u64>,
    ) -> Result<WorkPlaneEvaluation, FeatureReplayError> {
        let range = FeatureReplayRange {
            start: 0,
            end: self.effective_end(),
        };
        self.validate_work_plane_reads()?;
        let mut controller =
            FeatureReplayController::new_with_parameters(self, range, vec![], parameters)?;
        let mut evaluation = WorkPlaneEvaluation::default();
        evaluation.replay(self, parameters, &mut controller)?;
        evaluation.health = controller.finish()?;
        Ok(evaluation)
    }

    pub(super) fn validate_work_plane_reads(&self) -> Result<(), FeatureReplayError> {
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

impl WorkPlaneEvaluation {
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
        let feature = sequence
            .features()
            .iter()
            .find(|feature| feature.id() == id)
            .ok_or(WorkPlaneError::Unavailable(id))?;
        let plane = feature.definition().evaluate(parameters, self)?;
        self.planes.insert(id, plane);
        Ok(())
    }
}
