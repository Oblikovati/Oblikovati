// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::RecipeIdAllocator,
    sketch::{Sketch2D, SketchEntityId, SketchPlacementRecipe, SketchPlane},
};
pub(super) fn build(
    recipe: &SketchPlacementRecipe,
) -> Result<(Sketch2D, Vec<SketchEntityId>), String> {
    let sketch = DocumentSketchId(1);
    let mut history = scratch_history(sketch)?;
    let token = history
        .begin("Build transient preview")
        .map_err(|cause| cause.to_string())?;
    // This owner is private and discarded. No application document or allocator is borrowed.
    let report = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_placement(sketch, recipe, &[], Default::default(), ids)
        })
        .map_err(|cause| cause.to_string())?;
    let geometry = history
        .snapshot()
        .sketches
        .get(sketch)
        .map_err(|cause| cause.to_string())?
        .geometry()
        .clone();
    Ok((geometry, report.members))
}

fn scratch_history(sketch: DocumentSketchId) -> Result<DocumentEditHistory, String> {
    let mut snapshot = empty_snapshot();
    snapshot
        .sketches
        .add(
            sketch,
            "Transient preview",
            SketchPlane::xy(),
            Sketch2D::default(),
        )
        .map_err(|cause| cause.to_string())?;
    DocumentEditHistory::new(snapshot, RecipeIdAllocator::default())
        .map_err(|cause| cause.to_string())
}

fn empty_snapshot() -> DocumentSnapshot {
    DocumentSnapshot {
        work_geometry: Default::default(),
        sketches: Default::default(),
        attributes: Default::default(),
        attachments: Default::default(),
        interests: Default::default(),
        resources: Default::default(),
        assets: Default::default(),
        body_names: Default::default(),
        sketch_settings: None,
        parameters: oblikovati_model::parameter::ParameterCollection::new(Default::default()),
        properties: Default::default(),
    }
}
