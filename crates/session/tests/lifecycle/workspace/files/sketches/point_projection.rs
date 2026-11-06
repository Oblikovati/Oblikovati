// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::SketchPointProjection,
    feature::{
        AuthoredFeature, FeatureId,
        work_geometry::{DatumLength, WorkGeometryDefinition, WorkPointReference},
    },
    sketch::SketchEntityId,
};
#[derive(Clone, Copy)]
struct ProjectedPointIds {
    sketch: DocumentSketchId,
    datum: FeatureId,
    anchor: SketchEntityId,
    driver: u64,
}
struct ProjectedWorkspace {
    host: Host,
    source: SequenceGuids,
    document: u64,
    ids: ProjectedPointIds,
}
fn saved_projection() -> ProjectedWorkspace {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let document = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let ids = host
        .commands(document)
        .unwrap()
        .run("Projected source", add_projection)
        .unwrap();
    host.save(document, "alpha", &mut source).unwrap();
    ProjectedWorkspace {
        host,
        source,
        document,
        ids,
    }
}
fn add_projection(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
) -> Result<ProjectedPointIds, String> {
    let reserved = reserve_projection(snapshot, ids)?;
    install_projection(snapshot, reserved)?;
    Ok(reserved)
}
fn reserve_projection(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
) -> Result<ProjectedPointIds, String> {
    let driver = reserve_driver(snapshot, ids)?;
    let datum = reserve_point_source(snapshot, ids, driver)?;
    let sketch = ids.allocate_sketch().map_err(|e| e.to_string())?;
    let anchor = ids.allocate_sketch_entity().map_err(|e| e.to_string())?;
    Ok(ProjectedPointIds {
        sketch,
        datum,
        anchor,
        driver,
    })
}
fn reserve_driver(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
) -> Result<u64, String> {
    let driver = ids.allocate().map_err(|e| e.to_string())?;
    snapshot
        .parameters
        .insert_expression(driver, "projected_height", USER_PARAM, "4 cm")
        .map_err(|e| e.to_string())?;
    Ok(driver)
}
fn reserve_point_source(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
    driver: u64,
) -> Result<FeatureId, String> {
    let datum = ids.allocate_feature().map_err(|e| e.to_string())?;
    let definition = WorkGeometryDefinition::Point([
        DatumLength::Constant(2.0),
        DatumLength::Parameter(driver),
        DatumLength::Constant(0.0),
    ]);
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(datum, "Source", definition))
        .map_err(|e| e.to_string())?;
    Ok(datum)
}
fn install_projection(
    snapshot: &mut DocumentSnapshot,
    ids: ProjectedPointIds,
) -> Result<(), String> {
    let mut geometry = Sketch2D::default();
    geometry
        .add_point(ids.anchor, [99.0; 2])
        .map_err(|e| e.to_string())?;
    snapshot
        .sketches
        .add(ids.sketch, "Projected", SketchPlane::xy(), geometry)
        .map_err(|e| e.to_string())?;
    let binding = SketchPointProjection {
        anchor: ids.anchor,
        reference: WorkPointReference::Feature(ids.datum),
    };
    snapshot
        .sketches
        .set_point_projections(ids.sketch, vec![binding])
        .map_err(|e| e.to_string())
}
fn snapshot(workspace: &ProjectedWorkspace) -> &DocumentSnapshot {
    workspace
        .host
        .workspace()
        .session(workspace.document)
        .unwrap()
        .snapshot()
}
fn assert_ready(workspace: &ProjectedWorkspace, height: f64) {
    let snapshot = snapshot(workspace);
    let view = snapshot
        .require_resolved_sketch(workspace.ids.sketch)
        .unwrap();
    assert_eq!(
        view.sketch
            .geometry()
            .point(workspace.ids.anchor)
            .unwrap()
            .position,
        [2.0, height]
    );
    assert_eq!(view.verification.structure.degrees_of_freedom, 0);
    assert_eq!(
        view.sketch.point_projections()[0].reference,
        WorkPointReference::Feature(workspace.ids.datum)
    );
}
fn edit_height(workspace: &mut ProjectedWorkspace, source: &str) {
    let driver = workspace.ids.driver;
    workspace
        .host
        .commands(workspace.document)
        .unwrap()
        .run("Source height", |snapshot, _| {
            snapshot.parameters.set_expression(driver, source)
        })
        .unwrap();
}
fn reopen(workspace: &mut ProjectedWorkspace) {
    workspace
        .host
        .close_discarding_changes(workspace.document)
        .unwrap();
    workspace.document = open_file(
        &mut workspace.host,
        "a.obk",
        &mut workspace.source,
        OpenActivation::Activate,
    );
}
fn undo_projected_edit(workspace: &mut ProjectedWorkspace) {
    workspace
        .host
        .commands(workspace.document)
        .unwrap()
        .undo()
        .unwrap();
}
fn redo_projected_edit(workspace: &mut ProjectedWorkspace) {
    workspace
        .host
        .commands(workspace.document)
        .unwrap()
        .redo()
        .unwrap();
}
fn save_projection(workspace: &mut ProjectedWorkspace) {
    workspace
        .host
        .save(workspace.document, "alpha", &mut workspace.source)
        .unwrap();
}
fn analyze_projection(
    workspace: &ProjectedWorkspace,
) -> oblikovati_solve::diagnostics::ConstraintStructure {
    workspace
        .host
        .workspace()
        .session(workspace.document)
        .unwrap()
        .analyze_sketch(workspace.ids.sketch)
        .unwrap()
}
fn selected_projection_report(
    workspace: &mut ProjectedWorkspace,
) -> oblikovati_model::document_sketch::DocumentSketchRecompute {
    workspace
        .host
        .commands(workspace.document)
        .unwrap()
        .solve_sketch(workspace.ids.sketch, || false)
        .unwrap()
}
fn edit_stored_driver(workspace: &ProjectedWorkspace, expression: &str) {
    let bytes = workspace.host.files().contents.borrow()[&path("a.obk")].clone();
    let package = DocumentPackage::decode(&bytes).unwrap();
    let mut restored = package.restore_document_parameters().unwrap();
    restored
        .parameters
        .set_expression(workspace.ids.driver, expression)
        .unwrap();
    let changed = package
        .with_parameters(&restored.parameters, restored.parameters.units().clone())
        .unwrap();
    workspace
        .host
        .files()
        .contents
        .borrow_mut()
        .insert(path("a.obk"), changed.encode().unwrap());
}
#[test]
fn reopen_resolves_changed_stored_drivers_before_accepting_verification_or_saved_seed() {
    let mut workspace = saved_projection();
    edit_stored_driver(&workspace, "9 cm");
    reopen(&mut workspace);
    assert_ready(&workspace, 9.0);
    let session = workspace
        .host
        .workspace()
        .session(workspace.document)
        .unwrap();
    assert!(session.undo_labels().is_empty());
    assert!(session.redo_labels().is_empty());
    assert!(
        workspace
            .host
            .workspace()
            .needs_save(workspace.document)
            .unwrap()
    );
}

#[test]
fn automatic_projection_recompute_undo_redo_and_save_reopen_keep_source_and_anchor_ids() {
    let mut workspace = saved_projection();
    assert_ready(&workspace, 4.0);
    edit_height(&mut workspace, "6 cm");
    assert_ready(&workspace, 6.0);
    undo_projected_edit(&mut workspace);
    assert_ready(&workspace, 4.0);
    redo_projected_edit(&mut workspace);
    assert_ready(&workspace, 6.0);
    save_projection(&mut workspace);
    reopen(&mut workspace);
    assert_ready(&workspace, 6.0);
    assert!(
        !workspace
            .host
            .workspace()
            .needs_save(workspace.document)
            .unwrap()
    );
}
#[test]
fn selected_solve_and_analysis_use_current_sources_and_source_loss_recovers_on_edit() {
    let mut workspace = saved_projection();
    let structure = analyze_projection(&workspace);
    assert_eq!(structure.degrees_of_freedom, 0);
    edit_height(&mut workspace, "missing");
    assert!(
        snapshot(&workspace)
            .require_resolved_sketch(workspace.ids.sketch)
            .is_err()
    );
    let report = selected_projection_report(&mut workspace);
    assert!(report.reports[0].result.is_err());
    edit_height(&mut workspace, "8 cm");
    assert_ready(&workspace, 8.0);
}
#[test]
fn aborted_edits_discard_projected_coordinates_and_verification_together() {
    let mut workspace = saved_projection();
    let before = snapshot(&workspace).sketches.clone();
    workspace
        .host
        .commands(workspace.document)
        .unwrap()
        .begin("Abort source")
        .unwrap();
    edit_height(&mut workspace, "7 cm");
    assert_ready(&workspace, 7.0);
    workspace
        .host
        .commands(workspace.document)
        .unwrap()
        .abort()
        .unwrap();
    assert!(before.same_evaluated_state(&snapshot(&workspace).sketches));
}
#[test]
fn failed_edits_discard_projected_coordinates_and_verification_together() {
    let mut workspace = saved_projection();
    let before = snapshot(&workspace).sketches.clone();
    let driver = workspace.ids.driver;
    let result = workspace
        .host
        .commands(workspace.document)
        .unwrap()
        .run::<(), String>("Fail source", |snapshot, _| {
            snapshot.parameters.set_expression(driver, "9 cm").unwrap();
            Err("declined".into())
        });
    assert!(result.is_err());
    assert!(before.same_evaluated_state(&snapshot(&workspace).sketches));
}
