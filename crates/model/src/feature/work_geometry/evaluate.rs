// SPDX-License-Identifier: GPL-2.0-only

use super::{WorkGeometryDefinition as WorkPlaneDefinition, WorkGeometryError as WorkPlaneError};
use crate::feature::work_plane::WorkPlaneResolver;
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
pub struct WorkGeometryEvaluation {
    planes: BTreeMap<FeatureId, SketchPlane>,
    axes: BTreeMap<FeatureId, crate::feature::work_axis::WorkAxisLine>,
    pub health: Vec<FeatureHealthRecord<WorkPlaneError>>,
}

impl WorkPlaneResolver for WorkGeometryEvaluation {
    fn plane(&self, feature: FeatureId) -> Result<SketchPlane, WorkPlaneError> {
        if self.axes.contains_key(&feature) {
            return Err(WorkPlaneError::Kind {
                feature,
                expected: "plane",
                actual: "axis",
            });
        }
        self.planes
            .get(&feature)
            .copied()
            .ok_or(WorkPlaneError::Unavailable(feature))
    }
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
        let feature = sequence
            .get(id)
            .map_err(|_| WorkPlaneError::Unavailable(id))?;
        match *feature.definition() {
            WorkPlaneDefinition::Axis(axis) => {
                self.axes.insert(id, axis);
            }
            definition => {
                let plane = definition.evaluate(parameters, self)?;
                self.planes.insert(id, plane);
            }
        }
        Ok(())
    }
}

impl WorkGeometryEvaluation {
    /// Resolve one available current axis by stable ID; example: `evaluation.axis(FeatureId(3))?`.
    pub fn axis(
        &self,
        feature: FeatureId,
    ) -> Result<crate::feature::work_axis::WorkAxisLine, WorkPlaneError> {
        if self.planes.contains_key(&feature) {
            return Err(WorkPlaneError::Kind {
                feature,
                expected: "axis",
                actual: "plane",
            });
        }
        self.axes.get(&feature).copied().ok_or_else(|| {
            WorkPlaneError::Frame(crate::sketch::SketchError::Definition(format!(
                "datum {feature:?}: expected an available current axis"
            )))
        })
    }
}
