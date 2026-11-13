// SPDX-License-Identifier: GPL-2.0-only
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::DocumentSnapshot,
    sketch::{Sketch2D, SketchEntityId, SketchError, SketchPoint2D, SketchSelectionTransform},
};
use oblikovati_solve::nonlinear::{SolveOptions, SolveTermination};
/// Solved private geometry and the selected entities for a transform preview.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchTransformPreview {
    pub geometry: Sketch2D,
    pub members: Vec<SketchEntityId>,
}
/// Authored defining points or an input decline before adoption.
/// Example: `let reply: SketchTransformProjectionInput = Ok(points);`.
pub type SketchTransformProjectionInput = Result<Vec<SketchEntityId>, SketchError>;
impl<O: TransactionObserver> DocumentSession<O> {
    /// Preview a transform with the same recompute as commit, without changing history.
    /// Example: `session.preview_sketch_transform(sketch, &[line], transform)?`.
    pub fn preview_sketch_transform(
        &self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        transform: SketchSelectionTransform,
    ) -> Result<SketchTransformPreview, CommandError<SketchError>> {
        let options = self.sketch_options;
        self.preview_projected(
            |snapshot, _| apply_pinned_transform(snapshot, sketch, entities, transform, options),
            |snapshot, _| {
                Ok(SketchTransformPreview {
                    geometry: snapshot.sketches.get(sketch)?.geometry().clone(),
                    members: entities.to_vec(),
                })
            },
        )
    }
    /// Move selected defining points through history and current constraint/projection recompute.
    /// Example: `session.translate_sketch_entities(sketch, &[line], [2.0, 3.0])?;`.
    pub fn translate_sketch_entities(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        vector: [f64; 2],
    ) -> Result<Vec<SketchEntityId>, CommandError<SketchError>> {
        self.translate_sketch_entities_projected(sketch, entities, vector, |result| result)
    }
    /// Prepare the response before adopting coordinates; reply errors discard the candidate.
    /// Example: `session.translate_sketch_entities_projected(sketch, &[line], [2.0, 3.0], |result| encode(result?))?;`.
    pub fn translate_sketch_entities_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        vector: [f64; 2],
        project: impl FnOnce(SketchTransformProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.transform_sketch_entities_projected(
            sketch,
            entities,
            SketchSelectionTransform::Move(vector),
            project,
        )
    }
    /// Prepare a rigid edit response before adoption; example: `session.transform_sketch_entities_projected(sketch, &entities, transform, |result| result)?;`.
    pub fn transform_sketch_entities_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        transform: SketchSelectionTransform,
        project: impl FnOnce(SketchTransformProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        let options = self.sketch_options;
        self.run(transform.label(), |snapshot, _| {
            project(apply_pinned_transform(
                snapshot, sketch, entities, transform, options,
            ))
        })
    }
}

fn apply_pinned_transform(
    snapshot: &mut DocumentSnapshot,
    sketch: DocumentSketchId,
    entities: &[SketchEntityId],
    transform: SketchSelectionTransform,
    options: SolveOptions,
) -> Result<Vec<SketchEntityId>, SketchError> {
    let mut geometry = snapshot.sketches.get(sketch)?.geometry().clone();
    let selected = geometry.transform_entities(entities, transform)?;
    let anchors = transform_anchors(snapshot, sketch, &geometry, &selected, transform)?;
    let solved =
        geometry.solve_with_grounded_points(&snapshot.parameters, &anchors, options, || false)?;
    if solved.numerical().termination != SolveTermination::Converged {
        return Err(SketchError::Definition(format!(
            "transform {transform:?} selection {entities:?}: expected a converged move under the current constraints; solver returned {:?}",
            solved.numerical().termination
        )));
    }
    snapshot
        .sketches
        .replace_geometry(sketch, solved.into_candidate())?;
    Ok(selected)
}
fn transform_anchors(
    snapshot: &DocumentSnapshot,
    sketch: DocumentSketchId,
    geometry: &Sketch2D,
    selected: &[SketchEntityId],
    transform: SketchSelectionTransform,
) -> Result<Vec<SketchPoint2D>, SketchError> {
    let mut anchors = snapshot.resolve_sketch_projected_points(sketch)?;
    for id in selected {
        let target = *geometry.point(*id)?;
        if let Some(reference) = anchors.iter().find(|anchor| anchor.id == *id) {
            if reference.position != target.position {
                return Err(SketchError::Definition(format!(
                    "transform {transform:?} point {id:?}: expected its projected position {:?}, received {:?}",
                    reference.position, target.position
                )));
            }
        } else {
            anchors.push(target);
        }
    }
    Ok(anchors)
}
