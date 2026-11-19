// SPDX-License-Identifier: GPL-2.0-only
//! Bind concrete datum evaluations to the model inputs that produced them (#4107).
use super::*;
use crate::{document_sketch::DocumentSketchId, history::DocumentSnapshot, sketch::SketchError};
use std::sync::Arc;

#[derive(Clone, Debug)]
pub(super) enum EvaluationSource {
    Datum {
        program: FeatureSequence<WorkPlaneDefinition>,
        parameters: ParameterCollection<u64>,
    },
    Part {
        input: Arc<DocumentSnapshot>,
        output: Arc<DocumentSnapshot>,
    },
}
impl WorkGeometryEvaluation {
    pub(super) fn bind_datum_source(
        &mut self,
        program: &FeatureSequence<WorkPlaneDefinition>,
        parameters: &ParameterCollection<u64>,
    ) {
        self.source_evidence = Some(EvaluationSource::Datum {
            program: program.clone(),
            parameters: parameters.clone(),
        });
    }
    pub(crate) fn bind_part_source(&mut self, snapshot: Arc<DocumentSnapshot>) {
        self.bind_part_sources(Arc::clone(&snapshot), snapshot);
    }
    pub(crate) fn bind_part_sources(
        &mut self,
        input: Arc<DocumentSnapshot>,
        output: Arc<DocumentSnapshot>,
    ) {
        self.source_evidence = Some(EvaluationSource::Part { input, output });
    }
    pub(crate) fn has_part_source_for(&self, source: &Arc<DocumentSnapshot>) -> bool {
        matches!(&self.source_evidence, Some(EvaluationSource::Part { output, .. }) if Arc::ptr_eq(output, source))
    }
    pub(crate) fn require_frame_source(
        &self,
        snapshot: &DocumentSnapshot,
        sketch: DocumentSketchId,
    ) -> Result<(), SketchError> {
        let current = match &self.source_evidence {
            Some(EvaluationSource::Datum {
                program,
                parameters,
            }) => {
                program.same_work_plane_recipe(&snapshot.effective_work_geometry())
                    && parameters.same_recipe(&snapshot.parameters)
            }
            Some(EvaluationSource::Part { input, output }) => {
                same_part_source(output, snapshot, sketch)
                    || (!Arc::ptr_eq(input, output) && same_part_source(input, snapshot, sketch))
            }
            None => false,
        };
        if !current {
            return Err(SketchError::Definition(format!(
                "sketch {sketch:?} datum evaluation: expected model inputs from the current replay"
            )));
        }
        Ok(())
    }
    pub(crate) fn require_model_source(
        &self,
        snapshot: &DocumentSnapshot,
    ) -> Result<(), SketchError> {
        let current = match &self.source_evidence {
            Some(EvaluationSource::Datum {
                program,
                parameters,
            }) => {
                program.same_work_plane_recipe(&snapshot.effective_work_geometry())
                    && parameters.same_recipe(&snapshot.parameters)
            }
            Some(EvaluationSource::Part { input, output }) => {
                same_part_inputs(output, snapshot)
                    || (!Arc::ptr_eq(input, output) && same_part_inputs(input, snapshot))
            }
            None => false,
        };
        if !current {
            return Err(SketchError::Definition(
                "datum evaluation: expected the current model inputs".into(),
            ));
        }
        Ok(())
    }
}
fn same_part_inputs(source: &DocumentSnapshot, snapshot: &DocumentSnapshot) -> bool {
    source.parameters.same_recipe(&snapshot.parameters)
        && source
            .work_geometry
            .same_work_plane_recipe(&snapshot.work_geometry)
        && source.model_states == snapshot.model_states
        && source.sketches.same_recipe(&snapshot.sketches)
}
fn same_part_source(
    source: &DocumentSnapshot,
    snapshot: &DocumentSnapshot,
    sketch: DocumentSketchId,
) -> bool {
    if !source.parameters.same_recipe(&snapshot.parameters)
        || !source
            .work_geometry
            .same_work_plane_recipe(&snapshot.work_geometry)
        || source.model_states != snapshot.model_states
    {
        return false;
    }
    if source.sketches.same_recipe(&snapshot.sketches) {
        return true;
    }
    if source.sketches.get(sketch).is_ok() {
        return false;
    }
    let mut existing = snapshot.sketches.clone();
    existing.remove(sketch).is_ok() && source.sketches.same_recipe(&existing)
}
