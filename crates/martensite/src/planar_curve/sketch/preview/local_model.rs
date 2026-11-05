// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::RecipeIdAllocator,
    sketch::{Sketch2D, SketchEntityId, SketchPlacementRecipe, SketchPlane},
};
pub(super) fn build(
    recipe: &SketchPlacementRecipe,
    options: oblikovati_solve::nonlinear::SolveOptions,
) -> Result<(Sketch2D, Vec<SketchEntityId>), String> {
    let sketch = DocumentSketchId(1);
    let mut history = scratch_history(sketch)?;
    let report = append_recipe(&mut history, sketch, recipe, options)?;
    let geometry = solved_geometry(history.snapshot(), sketch, options)?;
    recipe
        .verify_circular_seeds(
            &geometry,
            &report.members,
            Some(&crate::planar_curve::MartensiteSketchConstruction),
        )
        .map_err(|cause| cause.to_string())?;
    Ok((geometry, report.members))
}

fn append_recipe(
    history: &mut DocumentEditHistory,
    sketch: DocumentSketchId,
    recipe: &SketchPlacementRecipe,
    options: oblikovati_solve::nonlinear::SolveOptions,
) -> Result<oblikovati_model::history::SketchPlacementCreationReport, String> {
    let token = history
        .begin("Build transient preview")
        .map_err(|cause| cause.to_string())?;
    // This owner is private and discarded. No application document or allocator is borrowed.
    history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_placement(sketch, recipe, &[], options, ids)
        })
        .map_err(|cause| cause.to_string())
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
fn solved_geometry(
    snapshot: &DocumentSnapshot,
    sketch: DocumentSketchId,
    options: oblikovati_solve::nonlinear::SolveOptions,
) -> Result<Sketch2D, String> {
    let geometry = snapshot
        .sketches
        .get(sketch)
        .map_err(|cause| cause.to_string())?
        .geometry();
    // Go aa8297d1 commitRecipeEntities solves intrinsic circularity as well as authored relations.
    let solved = geometry
        .solve(&snapshot.parameters, options, || false)
        .map_err(|cause| format!("preview solve: {cause}"))?;
    if solved.numerical().termination != oblikovati_solve::nonlinear::SolveTermination::Converged {
        return Err(format!(
            "preview solve {:?}: expected converged placement geometry",
            solved.numerical().termination
        ));
    }
    Ok(solved.into_candidate())
}
